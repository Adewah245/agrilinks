package repository

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/models"
)

const virtualAccountColumns = `id,user_id,provider,tx_ref,order_ref,provider_ref,account_number,bank_name,account_name,currency,status,identity_kind,consent_version,failure_reason,COALESCE(consent_at,'epoch'::timestamp),COALESCE(last_checked_at,'epoch'::timestamp),created_at,updated_at`

func scanVirtualAccount(row interface{ Scan(...any) error }, a *models.WalletVirtualAccount) error {
	return row.Scan(&a.ID,&a.UserID,&a.Provider,&a.TxRef,&a.OrderRef,&a.ProviderRef,&a.AccountNumber,&a.BankName,&a.AccountName,&a.Currency,&a.Status,&a.IdentityKind,&a.ConsentVersion,&a.FailureReason,&a.ConsentAt,&a.LastCheckedAt,&a.CreatedAt,&a.UpdatedAt)
}

// ClaimVirtualAccount serializes setup attempts across server instances without
// holding a database transaction open during a provider request.
func (r *WalletRepository) ClaimVirtualAccount(ctx context.Context, userID int, reference, kind, consent string, cooldown time.Duration) (*models.WalletVirtualAccount, bool, bool, error) {
	tx, err := r.db.BeginTx(ctx,nil)
	if err != nil { return nil,false,false,err }
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,`INSERT INTO wallet_virtual_accounts(user_id,tx_ref,identity_kind,consent_version,consent_at) VALUES($1,$2,$3,$4,CURRENT_TIMESTAMP) ON CONFLICT(user_id) DO NOTHING`,userID,reference,kind,consent)
	if err != nil { return nil,false,false,err }
	n, err := result.RowsAffected()
	if err != nil { return nil,false,false,err }
	a := &models.WalletVirtualAccount{}
	if err = scanVirtualAccount(tx.QueryRowContext(ctx,`SELECT `+virtualAccountColumns+` FROM wallet_virtual_accounts WHERE user_id=$1 FOR UPDATE`,userID),a); err != nil { return nil,false,false,err }
	created := n == 1
	if a.Ready() || (!created && a.Status == models.VirtualAccountPending && time.Since(a.UpdatedAt) < cooldown) {
		return a,false,created,tx.Commit()
	}
	_,err = tx.ExecContext(ctx,`UPDATE wallet_virtual_accounts SET status='pending',identity_kind=$2,consent_version=$3,consent_at=CURRENT_TIMESTAMP,failure_reason='',last_checked_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE user_id=$1`,userID,kind,consent)
	if err != nil { return nil,false,created,err }
	return a,true,created,tx.Commit()
}

func (r *WalletRepository) SaveVirtualAccount(ctx context.Context, a *models.WalletVirtualAccount) error {
	_,err := r.db.ExecContext(ctx,`UPDATE wallet_virtual_accounts SET order_ref=COALESCE(NULLIF($2,''),order_ref),provider_ref=COALESCE(NULLIF($3,''),provider_ref),account_number=COALESCE(NULLIF($4,''),account_number),bank_name=COALESCE(NULLIF($5,''),bank_name),account_name=COALESCE(NULLIF($6,''),account_name),currency=$7,status=$8,failure_reason=$9,last_checked_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE user_id=$1`,a.UserID,a.OrderRef,a.ProviderRef,a.AccountNumber,a.BankName,a.AccountName,a.Currency,a.Status,a.FailureReason)
	return err
}

func (r *WalletRepository) FailVirtualAccount(ctx context.Context, userID int, reason string) error {
	_,err := r.db.ExecContext(ctx,`UPDATE wallet_virtual_accounts SET status='failed',failure_reason=$2,last_checked_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND status<>'active'`,userID,reason)
	return err
}

func (r *WalletRepository) VirtualAccountUsers(ctx context.Context, limit int) ([]int,error) {
	rows,err := r.db.QueryContext(ctx,`SELECT user_id FROM wallet_virtual_accounts WHERE status='pending' ORDER BY last_checked_at NULLS FIRST,id LIMIT $1`,limit)
	if err != nil { return nil,err }
	defer rows.Close()
	ids := []int{}
	for rows.Next() { var id int; if err := rows.Scan(&id); err != nil { return nil,err }; ids = append(ids,id) }
	return ids,rows.Err()
}

func (r *WalletRepository) VirtualAccountByReference(ctx context.Context, reference string) (*models.WalletVirtualAccount,error) {
	a := &models.WalletVirtualAccount{}
	err := scanVirtualAccount(r.db.QueryRowContext(ctx,`SELECT `+virtualAccountColumns+` FROM wallet_virtual_accounts WHERE tx_ref=$1 AND provider='flutterwave'`,reference),a)
	if err == sql.ErrNoRows { return nil,nil }
	return a,err
}

func (r *WalletRepository) MarkVirtualAccountChecked(ctx context.Context, userID int) error {
	_,err := r.db.ExecContext(ctx,`UPDATE wallet_virtual_accounts SET last_checked_at=CURRENT_TIMESTAMP WHERE user_id=$1`,userID)
	return err
}
