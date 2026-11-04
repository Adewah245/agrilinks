import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text, formatDate, requestKey } from "../lib/view.js";
import mountFinance from "../behaviors/finance.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function PlatformRevenuePage({ data }) {
  usePageBehaviors([mountFinance], data);
  return (
    <>
      <main className="finance-page">
        <nav aria-label="Finance navigation">
          <a href="/admin/finance">Admin finance</a>
          <a href="/wallet">My wallet</a>
          <a href="#company-transfers">Company transfers</a>
          <a href="#financial-controls">Financial controls</a>
          <a href="#settlements">Settlements</a>
          <a href="#accounts">Chart of accounts</a>
        </nav>
        <p className="finance-eyebrow">
          Agrilinks Nigeria · FINANCIAL OPERATIONS
        </p>
        <h1>Platform revenue &amp; treasury</h1>
        <p>
          Revenue, customer obligations and reconciled company cash, with a
          traceable record of every movement.
        </p>
        {truth(data?.Error) ? (
          <>
            <p className="finance-alert" role="alert">
              {text(data?.Error)}
            </p>
          </>
        ) : null}
        {truth(data?.Notice) ? (
          <>
            <p className="finance-alert" role="status">
              {text(data?.Notice)}
            </p>
          </>
        ) : null}
        <section className="finance-card">
          <h2>Financial reporting</h2>
          <form
            className="finance-filters"
            method="GET"
            action="/admin/revenue"
          >
            <label>
              From (UTC)
              <input
                type="date"
                name="from"
                defaultValue={text(data?.ReportFrom)}
              />
            </label>
            <label>
              Through (UTC)
              <input
                type="date"
                name="to"
                defaultValue={text(data?.ReportTo)}
              />
            </label>
            <button className="finance-button">Update report</button>
            <button className="finance-button" name="export" value="ledger">
              Export ledger CSV
            </button>
          </form>
          <p>
            All-time balances appear alongside the selected period's revenue and
            expenses. Exports include every journal line, source, reference,
            account and amount in kobo.
          </p>
        </section>
        <section className="finance-metrics" aria-label="Accounting metrics">
          {truth(data?.Metrics)
            ? items(data?.Metrics).map((item1, index1) => (
                <Fragment key={index1}>
                  <article className="finance-card finance-metric">
                    <h2>{text(item1?.Label)}</h2>
                    <strong>{text(item1?.Value)}</strong>
                    <p>{text(item1?.Detail)}</p>
                  </article>
                </Fragment>
              ))
            : null}
        </section>
        <section className="finance-card" id="company-transfers">
          <h2>Company revenue transfers</h2>
          <p>
            Only reconciled, settled company cash remaining after customer
            liabilities, provider obligations and reserves can be transferred. A
            second administrator approves the request. Execution and confirmed
            bank receipt are separate steps.
          </p>
          {truth(data?.CompanyAccount) ? (
            <>
              <div className="finance-summary">
                <div>
                  <span>Source</span>
                  <strong>Settled provider cash</strong>
                </div>
                <div>
                  <span>Business destination</span>
                  <strong>{text(data?.CompanyAccount?.CompanyName)}</strong>
                  <p>
                    {text(data?.CompanyAccount?.BankName)} ·{" "}
                    {text(data?.CompanyAccount?.AccountNumber)}
                    <br />
                    {text(data?.CompanyAccount?.AccountName)}
                  </p>
                </div>
              </div>
            </>
          ) : (
            <>
              <p className="finance-alert">
                An approved company business account is required before
                requesting transfers. Propose the destination under Financial
                controls.
              </p>
            </>
          )}
          {truth(data?.CanRequest) ? (
            <>
              <form method="POST" className="finance-form">
                <input
                  type="hidden"
                  name="csrf"
                  defaultValue={text(data?.CSRF)}
                />
                <input type="hidden" name="action" defaultValue="quote" />
                <label>
                  Requested bank payout (NGN)
                  <input
                    name="amount"
                    type="number"
                    min="0.01"
                    step="0.01"
                    required={true}
                  />
                </label>
                <button>Review company transfer</button>
              </form>
            </>
          ) : null}
          {truth(data?.Quote) ? (
            <>
              <div className="finance-card finance-quote">
                <h3>Review before reserving funds</h3>
                <dl>
                  <div>
                    <dt>Bank receives</dt>
                    <dd>₦{text(data?.Quote?.Amount)}</dd>
                  </div>
                  <div>
                    <dt>Provider transfer charge</dt>
                    <dd>₦{text(data?.Quote?.ProviderFee)}</dd>
                  </div>
                  <div className="finance-total">
                    <dt>Total company cash reserved</dt>
                    <dd>₦{text(data?.Quote?.Total)}</dd>
                  </div>
                </dl>
                <p>
                  Quote expires{" "}
                  {text(formatDate(data?.Quote?.ExpiresAt, "15:04 MST"))}. This
                  reserves funds and creates a request for a different
                  administrator.
                </p>
                <form method="POST" className="finance-form">
                  <input
                    type="hidden"
                    name="csrf"
                    defaultValue={text(data?.CSRF)}
                  />
                  <input type="hidden" name="action" defaultValue="request" />
                  <input
                    type="hidden"
                    name="quote_id"
                    defaultValue={text(data?.Quote?.ID)}
                  />
                  <label>
                    Business reason
                    <Textarea
                      name="reason"
                      minLength="5"
                      required={true}
                    ></Textarea>
                  </label>
                  <fieldset>
                    <legend>Confirm your administrator identity</legend>
                    <p>
                      <button type="button" data-finance-code="">
                        Send one-time finance code
                      </button>
                    </p>
                    <label>
                      Current password
                      <input
                        type="password"
                        name="current_password"
                        autoComplete="current-password"
                        required={true}
                      />
                    </label>
                    <label>
                      Transaction PIN
                      <input
                        name="transaction_pin"
                        type="password"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="off"
                        required={true}
                      />
                    </label>
                    <label>
                      One-time finance code
                      <input
                        name="code"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="one-time-code"
                        required={true}
                      />
                    </label>
                    <p data-code-status="" role="status">
                      Keep your PIN and verification code private.
                    </p>
                  </fieldset>
                  <button>Reserve funds and request approval</button>
                </form>
              </div>
            </>
          ) : null}
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Reference</th>
                  <th>Payout</th>
                  <th>Provider fee</th>
                  <th>Total reserved</th>
                  <th>Status</th>
                  <th>Initiator / approver</th>
                </tr>
              </thead>
              <tbody>
                {truth(data?.Withdrawals) ? (
                  items(data?.Withdrawals).map((item2, index2) => (
                    <Fragment key={index2}>
                      <tr>
                        <td>
                          {text(item2?.Reference)}
                          <br />
                          <small>{text(item2?.Reason)}</small>
                        </td>
                        <td>₦{text(item2?.Amount)}</td>
                        <td>₦{text(item2?.ProviderFee)}</td>
                        <td>₦{text(item2?.Total)}</td>
                        <td>{text(item2?.Status)}</td>
                        <td>
                          {text(item2?.InitiatedBy)} /{" "}
                          {truth(item2?.ApprovedBy?.Valid) ? (
                            <>{text(item2?.ApprovedBy?.Int64)}</>
                          ) : (
                            <>Awaiting review</>
                          )}
                        </td>
                      </tr>
                    </Fragment>
                  ))
                ) : (
                  <>
                    <tr>
                      <td colSpan="6">No company transfers recorded.</td>
                    </tr>
                  </>
                )}
              </tbody>
            </table>
          </div>
          {truth(data?.CanApprove) ? (
            <>
              {truth(data?.Withdrawals)
                ? items(data?.Withdrawals).map((item3, index3) => (
                    <Fragment key={index3}>
                      {truth(
                        truth(item3?.Status === "pending") ||
                          truth(item3?.Status === "approved") ||
                          truth(item3?.Status === "processing"),
                      ) ? (
                        <>
                          <details className="finance-review">
                            <summary>
                              Review transfer #{text(item3?.ID)} · ₦
                              {text(item3?.Amount)} · {text(item3?.Status)}
                            </summary>
                            <p className="finance-reference">
                              {text(item3?.Reference)} · {text(item3?.Reason)} ·
                              Provider: {text(item3?.ProviderID)}
                            </p>
                            <form method="POST" className="finance-form">
                              <input
                                type="hidden"
                                name="csrf"
                                defaultValue={text(data?.CSRF)}
                              />
                              <input
                                type="hidden"
                                name="id"
                                defaultValue={text(item3?.ID)}
                              />
                              <label>
                                Action
                                <Select name="action">
                                  {truth(item3?.Status === "pending") ? (
                                    <>
                                      <option value="approve">
                                        Approve request
                                      </option>
                                      <option value="reject">
                                        Reject and release reservation
                                      </option>
                                    </>
                                  ) : truth(item3?.Status === "approved") ? (
                                    <>
                                      <option value="execute">
                                        Execute approved payment
                                      </option>
                                      <option value="reject">
                                        Cancel before execution
                                      </option>
                                    </>
                                  ) : (
                                    <>
                                      <option value="check">
                                        Verify provider outcome
                                      </option>
                                      <option value="confirm_receipt">
                                        Confirm exact business bank receipt
                                      </option>
                                    </>
                                  )}
                                </Select>
                              </label>
                              <label>
                                Decision / bank statement evidence
                                <Textarea
                                  name="reason"
                                  minLength="5"
                                  required={true}
                                ></Textarea>
                              </label>
                              {truth(item3?.Status === "processing") ? (
                                <>
                                  <label>
                                    Bank statement transaction reference
                                    <input name="statement_reference" />
                                  </label>
                                  <label>
                                    Amount actually received (NGN)
                                    <input
                                      name="amount"
                                      type="number"
                                      min="0.01"
                                      step="0.01"
                                    />
                                  </label>
                                </>
                              ) : null}
                              <fieldset>
                                <legend>
                                  Confirm your administrator identity
                                </legend>
                                <p>
                                  <button type="button" data-finance-code="">
                                    Send one-time finance code
                                  </button>
                                </p>
                                <label>
                                  Current password
                                  <input
                                    type="password"
                                    name="current_password"
                                    autoComplete="current-password"
                                    required={true}
                                  />
                                </label>
                                <label>
                                  Transaction PIN
                                  <input
                                    name="transaction_pin"
                                    type="password"
                                    inputMode="numeric"
                                    pattern="[0-9]{6}"
                                    maxLength="6"
                                    autoComplete="off"
                                    required={true}
                                  />
                                </label>
                                <label>
                                  One-time finance code
                                  <input
                                    name="code"
                                    inputMode="numeric"
                                    pattern="[0-9]{6}"
                                    maxLength="6"
                                    autoComplete="one-time-code"
                                    required={true}
                                  />
                                </label>
                                <p data-code-status="" role="status">
                                  Keep your PIN and verification code private.
                                </p>
                              </fieldset>
                              <button>Record financial action</button>
                            </form>
                          </details>
                        </>
                      ) : null}
                    </Fragment>
                  ))
                : null}
            </>
          ) : null}
        </section>
        <section className="finance-card" id="financial-controls">
          <h2>Financial controls &amp; approvals</h2>
          <p>
            Fee policies, bank destinations, limits, reserve allocations and
            assessed statutory obligations require a different authorised
            approver. Business destinations and opening reconciliations require
            a super administrator. Tax amounts require external accounting
            advice and supporting evidence.
          </p>
          {truth(data?.CanRequest) ? (
            <>
              <details>
                <summary>Propose a financial change</summary>
                <form method="POST" className="finance-form">
                  <input
                    type="hidden"
                    name="csrf"
                    defaultValue={text(data?.CSRF)}
                  />
                  <input type="hidden" name="action" defaultValue="change" />
                  <input
                    type="hidden"
                    name="request_key"
                    defaultValue={text(requestKey())}
                  />
                  <label>
                    Change type
                    <Select name="kind">
                      <option value="company_bank">
                        Company business bank destination
                      </option>
                      <option value="fee_policy">Fee policy</option>
                      <option value="transfer_limits">
                        Company transfer limits
                      </option>
                      <option value="reserve">
                        Reserve allocation or release
                      </option>
                      <option value="statutory_obligation">
                        Assessed statutory obligation
                      </option>
                      <option value="opening_reconciliation">
                        Historical opening cash reconciliation
                      </option>
                    </Select>
                  </label>
                  <details>
                    <summary>Business bank account</summary>
                    <label>
                      Company legal name
                      <input name="company_name" />
                    </label>
                    <label>
                      Bank name
                      <input name="bank_name" />
                    </label>
                    <label>
                      Provider bank code
                      <input name="bank_code" />
                    </label>
                    <label>
                      Business account number
                      <input
                        name="account_number"
                        inputMode="numeric"
                        pattern="[0-9]{10}"
                        maxLength="10"
                      />
                    </label>
                    <p>
                      The provider must resolve the account to the company legal
                      name. Include business ownership documents or their secure
                      reference in the evidence below.
                    </p>
                  </details>
                  <details>
                    <summary>Service fees and provider treatment</summary>
                    <label>
                      Funding fee (NGN)
                      <input
                        name="funding_fee"
                        type="number"
                        min="0"
                        step="0.01"
                        defaultValue="50"
                      />
                    </label>
                    <label>
                      Withdrawal ₦1–₦5,000 fee (NGN)
                      <input
                        name="withdrawal_low"
                        type="number"
                        min="0"
                        step="0.01"
                        defaultValue="50"
                      />
                    </label>
                    <label>
                      Withdrawal ₦5,001–₦9,999 fee (NGN)
                      <input
                        name="withdrawal_mid"
                        type="number"
                        min="0"
                        step="0.01"
                        defaultValue="200"
                      />
                    </label>
                    <label>
                      Withdrawal ₦10,000+ fee (NGN)
                      <input
                        name="withdrawal_high"
                        type="number"
                        min="0"
                        step="0.01"
                        defaultValue="300"
                      />
                    </label>
                    <label>
                      Withdrawal provider charges paid by
                      <Select name="provider_bearer">
                        <option value="customer">
                          Customer, disclosed on top
                        </option>
                        <option value="platform">Agrilinks Nigeria</option>
                      </Select>
                    </label>
                    <label>
                      Verified addition to provider fee quote (basis points)
                      <input
                        name="provider_tax_bps"
                        type="number"
                        min="0"
                        max="10000"
                        defaultValue="0"
                      />
                    </label>
                    <p>
                      Leave at zero unless merchant-specific documentation
                      confirms the quote excludes this charge. Provider fees
                      themselves come from the provider.
                    </p>
                    <label>
                      Effective from (UTC)
                      <input name="effective_at" type="datetime-local" />
                    </label>
                  </details>
                  <details>
                    <summary>Transfer limits</summary>
                    <label>
                      Per-transfer limit (NGN)
                      <input
                        name="per_transfer"
                        type="number"
                        min="1"
                        step="0.01"
                      />
                    </label>
                    <label>
                      Daily limit (NGN)
                      <input
                        name="daily_limit"
                        type="number"
                        min="1"
                        step="0.01"
                      />
                    </label>
                    <label>
                      Amount requiring super-admin approval (NGN)
                      <input
                        name="senior_threshold"
                        type="number"
                        min="1"
                        step="0.01"
                      />
                    </label>
                  </details>
                  <details>
                    <summary>
                      Reserves, statutory amounts and opening reconciliation
                    </summary>
                    <label>
                      Amount (NGN)
                      <input
                        name="amount"
                        type="number"
                        min="0.01"
                        step="0.01"
                      />
                    </label>
                    <label>
                      Account
                      <Select name="account">
                        <option value="company_operating_reserve">
                          Company operating reserve
                        </option>
                        <option value="chargeback_reserve">
                          Refund and chargeback reserve
                        </option>
                        <option value="provider_settled_cash">
                          Verified opening provider cash
                        </option>
                      </Select>
                    </label>
                    <label>
                      Reserve action
                      <Select name="release">
                        <option value="no">Allocate reserve</option>
                        <option value="yes">Release existing reserve</option>
                      </Select>
                    </label>
                    <label>
                      Opening statement reference
                      <input name="statement_reference" />
                    </label>
                  </details>
                  <label>
                    Supporting evidence / secure document references
                    <Textarea
                      name="evidence"
                      minLength="5"
                      required={true}
                    ></Textarea>
                  </label>
                  <label>
                    Reason for change
                    <Textarea
                      name="reason"
                      minLength="5"
                      required={true}
                    ></Textarea>
                  </label>
                  <fieldset>
                    <legend>Confirm your administrator identity</legend>
                    <p>
                      <button type="button" data-finance-code="">
                        Send one-time finance code
                      </button>
                    </p>
                    <label>
                      Current password
                      <input
                        type="password"
                        name="current_password"
                        autoComplete="current-password"
                        required={true}
                      />
                    </label>
                    <label>
                      Transaction PIN
                      <input
                        name="transaction_pin"
                        type="password"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="off"
                        required={true}
                      />
                    </label>
                    <label>
                      One-time finance code
                      <input
                        name="code"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="one-time-code"
                        required={true}
                      />
                    </label>
                    <p data-code-status="" role="status">
                      Keep your PIN and verification code private.
                    </p>
                  </fieldset>
                  <button>Submit for second-administrator approval</button>
                </form>
              </details>
            </>
          ) : null}
          {truth(data?.Changes) ? (
            items(data?.Changes).map((item4, index4) => (
              <Fragment key={index4}>
                <details className="finance-review">
                  <summary>
                    Change #{text(item4?.ID)} · {text(item4?.Kind)} ·{" "}
                    {text(item4?.Status)}
                  </summary>
                  <p>
                    Initiated by {text(item4?.InitiatedBy)} ·{" "}
                    {text(
                      formatDate(item4?.CreatedAt, "02 Jan 2006 15:04 UTC"),
                    )}
                  </p>
                  <p>{text(item4?.Reason)}</p>
                  <pre className="finance-evidence">{text(item4?.Payload)}</pre>
                  <p>{text(item4?.DecisionReason)}</p>
                  {truth(
                    truth(data?.CanApprove) &&
                      truth(item4?.Status === "pending"),
                  ) ? (
                    <>
                      <form method="POST" className="finance-form">
                        <input
                          type="hidden"
                          name="csrf"
                          defaultValue={text(data?.CSRF)}
                        />
                        <input
                          type="hidden"
                          name="id"
                          defaultValue={text(item4?.ID)}
                        />
                        <label>
                          Decision
                          <Select name="action">
                            <option value="approve_change">
                              Approve and apply change
                            </option>
                            <option value="reject_change">Reject change</option>
                          </Select>
                        </label>
                        <label>
                          Review reason
                          <Textarea
                            name="reason"
                            minLength="5"
                            required={true}
                          ></Textarea>
                        </label>
                        <fieldset>
                          <legend>Confirm your administrator identity</legend>
                          <p>
                            <button type="button" data-finance-code="">
                              Send one-time finance code
                            </button>
                          </p>
                          <label>
                            Current password
                            <input
                              type="password"
                              name="current_password"
                              autoComplete="current-password"
                              required={true}
                            />
                          </label>
                          <label>
                            Transaction PIN
                            <input
                              name="transaction_pin"
                              type="password"
                              inputMode="numeric"
                              pattern="[0-9]{6}"
                              maxLength="6"
                              autoComplete="off"
                              required={true}
                            />
                          </label>
                          <label>
                            One-time finance code
                            <input
                              name="code"
                              inputMode="numeric"
                              pattern="[0-9]{6}"
                              maxLength="6"
                              autoComplete="one-time-code"
                              required={true}
                            />
                          </label>
                          <p data-code-status="" role="status">
                            Keep your PIN and verification code private.
                          </p>
                        </fieldset>
                        <button>Record decision</button>
                      </form>
                    </>
                  ) : null}
                </details>
              </Fragment>
            ))
          ) : (
            <>
              <p>No proposed financial changes.</p>
            </>
          )}
        </section>
        <section className="finance-card" id="settlements">
          <h2>Provider &amp; bank settlements</h2>
          <p>
            Import provider records with their corresponding bank or
            provider-balance statement. Use Direction collection for incoming
            settlements, customer_payout for settled withdrawals, or
            provider_payable for paid provider invoices. A second administrator
            verifies the movements against the ledger. Mismatches remain
            excluded.
          </p>
          {truth(data?.CanRequest) ? (
            <>
              <details>
                <summary>Import normalized settlement statement</summary>
                <form method="POST" className="finance-form">
                  <input
                    type="hidden"
                    name="csrf"
                    defaultValue={text(data?.CSRF)}
                  />
                  <input
                    type="hidden"
                    name="action"
                    defaultValue="import_settlement"
                  />
                  <label>
                    Statement record (JSON; amounts in integer kobo)
                    <Textarea
                      name="statement"
                      rows="12"
                      spellCheck={false}
                      required={true}
                      placeholder={
                        '{"Provider":"flutterwave","Reference":"batch-reference","Currency":"NGN","Destination":"provider_balance","Source":"provider_statement","Evidence":"{\\"document\\":\\"secure-reference\\"}","StatementReference":"balance-statement-reference","GrossKobo":1005000,"DeductionsKobo":0,"NetKobo":1005000,"StatementKobo":1005000,"Items":[{"Reference":"payment-reference","GrossKobo":1005000,"DeductionsKobo":0,"NetKobo":1005000}]}'
                      }
                    ></Textarea>
                  </label>
                  <label>
                    Reconciliation reason
                    <Textarea
                      name="reason"
                      minLength="5"
                      required={true}
                    ></Textarea>
                  </label>
                  <fieldset>
                    <legend>Confirm your administrator identity</legend>
                    <p>
                      <button type="button" data-finance-code="">
                        Send one-time finance code
                      </button>
                    </p>
                    <label>
                      Current password
                      <input
                        type="password"
                        name="current_password"
                        autoComplete="current-password"
                        required={true}
                      />
                    </label>
                    <label>
                      Transaction PIN
                      <input
                        name="transaction_pin"
                        type="password"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="off"
                        required={true}
                      />
                    </label>
                    <label>
                      One-time finance code
                      <input
                        name="code"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="one-time-code"
                        required={true}
                      />
                    </label>
                    <p data-code-status="" role="status">
                      Keep your PIN and verification code private.
                    </p>
                  </fieldset>
                  <button>Import for independent reconciliation</button>
                </form>
              </details>
            </>
          ) : null}
          {truth(data?.Settlements) ? (
            items(data?.Settlements).map((item5, index5) => (
              <Fragment key={index5}>
                <details className="finance-review">
                  <summary>
                    Settlement #{text(item5?.ID)} · {text(item5?.Reference)} ·{" "}
                    {text(item5?.Status)}
                  </summary>
                  <p>
                    {text(item5?.Provider)} · {text(item5?.Direction)} ·{" "}
                    {text(item5?.Source)} · {text(item5?.Destination)} ·
                    Statement {text(item5?.StatementReference)}
                  </p>
                  <p>
                    Gross {text(item5?.GrossKobo)} kobo · Deductions{" "}
                    {text(item5?.DeductionsKobo)} kobo · Net{" "}
                    {text(item5?.NetKobo)} kobo
                  </p>
                  <pre className="finance-evidence">
                    {text(item5?.Evidence)}
                  </pre>
                  {truth(data?.CanApprove) ? (
                    <>
                      {truth(
                        truth(item5?.Status === "imported") ||
                          truth(item5?.Status === "mismatch") ||
                          truth(item5?.Status === "matched"),
                      ) ? (
                        <>
                          <form method="POST" className="finance-form">
                            <input
                              type="hidden"
                              name="csrf"
                              defaultValue={text(data?.CSRF)}
                            />
                            <input
                              type="hidden"
                              name="id"
                              defaultValue={text(item5?.ID)}
                            />
                            <input
                              type="hidden"
                              name="action"
                              defaultValue={
                                truth(item5?.Status !== "matched")
                                  ? "reconcile"
                                  : "reverse_settlement"
                              }
                            />
                            <label>
                              Statement review / reversal reason
                              <Textarea
                                name="reason"
                                minLength="5"
                                required={true}
                              ></Textarea>
                            </label>
                            <fieldset>
                              <legend>
                                Confirm your administrator identity
                              </legend>
                              <p>
                                <button type="button" data-finance-code="">
                                  Send one-time finance code
                                </button>
                              </p>
                              <label>
                                Current password
                                <input
                                  type="password"
                                  name="current_password"
                                  autoComplete="current-password"
                                  required={true}
                                />
                              </label>
                              <label>
                                Transaction PIN
                                <input
                                  name="transaction_pin"
                                  type="password"
                                  inputMode="numeric"
                                  pattern="[0-9]{6}"
                                  maxLength="6"
                                  autoComplete="off"
                                  required={true}
                                />
                              </label>
                              <label>
                                One-time finance code
                                <input
                                  name="code"
                                  inputMode="numeric"
                                  pattern="[0-9]{6}"
                                  maxLength="6"
                                  autoComplete="one-time-code"
                                  required={true}
                                />
                              </label>
                              <p data-code-status="" role="status">
                                Keep your PIN and verification code private.
                              </p>
                            </fieldset>
                            <button>
                              {truth(item5?.Status !== "matched") ? (
                                <>Match and approve settlement</>
                              ) : (
                                <>Record verified settlement reversal</>
                              )}
                            </button>
                          </form>
                        </>
                      ) : null}
                    </>
                  ) : null}
                </details>
              </Fragment>
            ))
          ) : (
            <>
              <p>No provider settlements imported.</p>
            </>
          )}
        </section>
        <section className="finance-card">
          <h2>Purchase protection review queue</h2>
          {truth(data?.Escrows) ? (
            items(data?.Escrows).map((item6, index6) => (
              <Fragment key={index6}>
                <p>
                  <a
                    href={"/orders/protection?order_id=" + text(item6?.OrderID)}
                  >
                    Order #{text(item6?.OrderID)} · {text(item6?.Products)}
                  </a>{" "}
                  — {text(item6?.Currency)}
                  {text(item6?.Amount)} · {text(item6?.State)} ·{" "}
                  {truth(item6?.Eligible) ? (
                    <>Eligible for review</>
                  ) : (
                    <>Awaiting required checks</>
                  )}
                </p>
              </Fragment>
            ))
          ) : (
            <>
              <p>No held purchases awaiting review.</p>
            </>
          )}
        </section>
        <section className="finance-card" id="accounts">
          <h2>Chart of accounts &amp; trial balance</h2>
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Account</th>
                  <th>Category</th>
                  <th>Debits (NGN)</th>
                  <th>Credits (NGN)</th>
                  <th>Debit balance (NGN)</th>
                </tr>
              </thead>
              <tbody>
                {truth(data?.Accounts)
                  ? items(data?.Accounts).map((item7, index7) => (
                      <Fragment key={index7}>
                        <tr>
                          <td>
                            {text(item7?.Name)}
                            <br />
                            <small>{text(item7?.Code)}</small>
                          </td>
                          <td>{text(item7?.Category)}</td>
                          <td>{text(item7?.Debit)}</td>
                          <td>{text(item7?.Credit)}</td>
                          <td>{text(item7?.Balance)}</td>
                        </tr>
                      </Fragment>
                    ))
                  : null}
              </tbody>
            </table>
          </div>
          <p>
            Credit balances display as negative here. Customer, seller, provider
            and company accounts remain separate.
          </p>
        </section>
        <footer>
          <p>Agrilinks Nigeria financial operations · NGN · UTC reporting</p>
        </footer>
      </main>
    </>
  );
}
