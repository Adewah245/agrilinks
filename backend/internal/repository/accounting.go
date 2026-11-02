package repository

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

type posting struct {
	Account       string
	Debit, Credit int64
}

func debit(account string, n int64) posting  { return posting{Account: account, Debit: n} }
func credit(account string, n int64) posting { return posting{Account: account, Credit: n} }
func walletAccount(id int64) string          { return fmt.Sprintf("wallet:%d", id) }
func ensureAccount(tx *sql.Tx, code, category string) error {
	var walletID any
	var id int64
	if _, err := fmt.Sscanf(code, "wallet:%d", &id); err == nil {
		walletID = id
	}
	_, err := tx.Exec(`INSERT INTO accounting_accounts(code,name,category,wallet_id) VALUES($1,$1,$2,$3) ON CONFLICT(code) DO NOTHING`, code, category, walletID)
	return err
}

// postJournal is a balanced, immutable destination record. The deferred SQL
// constraints independently prevent any unbalanced posting from committing.
func postJournal(tx *sql.Tx, reference, kind, currency string, actorID int, evidence string, lines ...posting) error {
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(721001)`); err != nil {
		return err
	}
	clean := []posting{}
	for _, line := range lines {
		if line.Debit == 0 && line.Credit == 0 {
			continue
		}
		if line.Debit < 0 || line.Credit < 0 || (line.Debit > 0 && line.Credit > 0) {
			return errors.New("invalid journal posting")
		}
		clean = append(clean, line)
	}
	if len(clean) == 0 {
		return nil
	}
	sort.Slice(clean, func(i, j int) bool {
		if clean[i].Account != clean[j].Account {
			return clean[i].Account < clean[j].Account
		}
		return clean[i].Debit < clean[j].Debit
	})
	body, _ := json.Marshal(struct {
		Kind, Currency string
		Lines          []posting
	}{kind, currency, clean})
	hash := sha256.Sum256(body)
	fingerprint := hex.EncodeToString(hash[:])
	if evidence == "" {
		evidence = "{}"
	}
	var reversalID any
	if kind == "reversal" {
		var metadata struct {
			Reference string `json:"reversal_of"`
		}
		if err := json.Unmarshal([]byte(evidence), &metadata); err != nil {
			return err
		}
		var previous int64
		if err := tx.QueryRow(`SELECT id FROM accounting_journals WHERE reference=$1`, metadata.Reference).Scan(&previous); err != nil {
			return err
		}
		reversalID = previous
	}
	var id int64
	err := tx.QueryRow(`INSERT INTO accounting_journals(reference,transaction_type,fingerprint,actor_id,evidence,reversal_of) VALUES($1,$2,$3,NULLIF($4,0),$5::jsonb,$6) ON CONFLICT(reference) DO NOTHING RETURNING id`, reference, kind, fingerprint, actorID, evidence, reversalID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var saved string
		if err = tx.QueryRow(`SELECT fingerprint FROM accounting_journals WHERE reference=$1`, reference).Scan(&saved); err != nil {
			return err
		}
		if saved != fingerprint {
			return errors.New("accounting reference reused with different postings")
		}
		return nil
	}
	if err != nil {
		return err
	}
	for i, line := range clean {
		if _, err = tx.Exec(`INSERT INTO accounting_postings(journal_id,line,account_code,currency,debit_kobo,credit_kobo) VALUES($1,$2,$3,$4,$5,$6)`, id, i+1, line.Account, currency, line.Debit, line.Credit); err != nil {
			return err
		}
	}
	return nil
}

func reverseJournal(tx *sql.Tx, reference, reversalRef, reason string) error {
	var id int64
	if err := tx.QueryRow(`SELECT id FROM accounting_journals WHERE reference=$1`, reference).Scan(&id); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT account_code,currency,debit_kobo,credit_kobo FROM accounting_postings WHERE journal_id=$1 ORDER BY line`, id)
	if err != nil {
		return err
	}
	var lines []posting
	currency := ""
	for rows.Next() {
		var p posting
		var c string
		if err = rows.Scan(&p.Account, &c, &p.Credit, &p.Debit); err != nil {
			rows.Close()
			return err
		}
		currency = c
		lines = append(lines, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{"reversal_of": reference, "reason": reason})
	// Original history is untouched. The reversal has its own stable reference.
	return postJournal(tx, reversalRef, "reversal", currency, 0, string(evidence), lines...)
}
