import { Select } from "../components/FormControls.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  withValue,
  text,
  humanPaymentStatus,
  humanStatus,
  formatDate,
  len,
  requestKey,
} from "../lib/view.js";
import mountWallet from "../behaviors/wallet.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function WalletPage({ data }) {
  usePageBehaviors([mountWallet], data);
  return (
    <>
      <a className="wallet-skip-link" href="#wallet-main">
        Skip to wallet
      </a>
      <svg
        className="wallet-icon-definitions"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <symbol id="w-shield" viewBox="0 0 24 24">
          <path d="M12 3 4 6v6c0 4 3.5 7 8 9 4.5-2 8-5 8-9V6Z" />
          <path d="M12 16V8m0 5c-4 0-5-3-5-3 4 0 5 3 5 3Zm0-2c4 0 5-3 5-3-4 0-5 3-5 3Z" />
        </symbol>
        <symbol id="w-grid" viewBox="0 0 24 24">
          <rect x="3" y="3" width="7" height="7" rx="1.5" />
          <rect x="14" y="3" width="7" height="7" rx="1.5" />
          <rect x="3" y="14" width="7" height="7" rx="1.5" />
          <rect x="14" y="14" width="7" height="7" rx="1.5" />
        </symbol>
        <symbol id="w-market" viewBox="0 0 24 24">
          <path d="m3 10 2-6h14l2 6M5 13v7h14v-7M3 10c0 4 4 4 4 0 0 4 5 4 5 0 0 4 5 4 5 0 0 4 4 4 4 0M10 20v-5h4v5" />
        </symbol>
        <symbol id="w-bag" viewBox="0 0 24 24">
          <path d="M5 7h14l1 14H4L5 7Zm3 0V6a4 4 0 0 1 8 0v1" />
        </symbol>
        <symbol id="w-wallet" viewBox="0 0 24 24">
          <path d="M20 8V5H5a2 2 0 0 0 0 4h15v11H5a2 2 0 0 1-2-2V7m17 5h-6v5h6" />
          <path d="M16 14.5h.01" />
        </symbol>
        <symbol id="w-card" viewBox="0 0 24 24">
          <rect x="3" y="5" width="18" height="14" rx="2" />
          <path d="M3 10h18M7 15h3" />
        </symbol>
        <symbol id="w-chat" viewBox="0 0 24 24">
          <path d="M21 11a9 9 0 0 1-13 8l-5 2 2-5A9 9 0 1 1 21 11Z" />
          <path d="M8 11h.01M12 11h.01M16 11h.01" />
        </symbol>
        <symbol id="w-arrow" viewBox="0 0 24 24">
          <path d="M5 12h14m-5-5 5 5-5 5" />
        </symbol>
        <symbol id="w-up-right" viewBox="0 0 24 24">
          <path d="M6 18 18 6M6 6h12v12" />
        </symbol>
        <symbol id="w-plus" viewBox="0 0 24 24">
          <path d="M12 5v14M5 12h14" />
        </symbol>
        <symbol id="w-send" viewBox="0 0 24 24">
          <path d="m21 3-7 18-4-7-7-4L21 3ZM10 14 21 3" />
        </symbol>
        <symbol id="w-bank" viewBox="0 0 24 24">
          <path d="m3 8 9-5 9 5H3Zm2 3v7m5-7v7m4-7v7m5-7v7M3 21h18" />
        </symbol>
        <symbol id="w-down" viewBox="0 0 24 24">
          <path d="M12 3v13m-5-5 5 5 5-5M4 17v4h16v-4" />
        </symbol>
        <symbol id="w-copy" viewBox="0 0 24 24">
          <rect x="8" y="8" width="12" height="13" rx="2" />
          <path d="M15 8V3H3v13h5" />
        </symbol>
        <symbol id="w-eye" viewBox="0 0 24 24">
          <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" />
          <circle cx="12" cy="12" r="3" />
        </symbol>
        <symbol id="w-eye-off" viewBox="0 0 24 24">
          <path d="m3 3 18 18M10 5a12 12 0 0 1 2 0c6.5 0 10 7 10 7a21 21 0 0 1-3 4M6 6a20 20 0 0 0-4 6s3.5 7 10 7a12 12 0 0 0 5-1M10 10a3 3 0 0 0 4 4" />
        </symbol>
        <symbol id="w-clock" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 7v5l3 2" />
        </symbol>
        <symbol id="w-bell" viewBox="0 0 24 24">
          <path d="M5 17h14l-2-3V9A5 5 0 0 0 7 9v5l-2 3Zm5 3h4M12 2v2" />
        </symbol>
        <symbol id="w-user" viewBox="0 0 24 24">
          <circle cx="12" cy="8" r="4" />
          <path d="M4 21v-2a8 8 0 0 1 16 0v2" />
        </symbol>
        <symbol id="w-logout" viewBox="0 0 24 24">
          <path d="M10 4H4v16h6m-1-8h12m-4-4 4 4-4 4" />
        </symbol>
        <symbol id="w-chevron" viewBox="0 0 24 24">
          <path d="m6 9 6 6 6-6" />
        </symbol>
        <symbol id="w-check" viewBox="0 0 24 24">
          <path d="m5 12 4 4L19 6" />
        </symbol>
        <symbol id="w-leaf" viewBox="0 0 24 24">
          <path d="M5 19C-1 9 10 3 21 3c0 11-6 22-16 16ZM4 21 16 9" />
        </symbol>
        <symbol id="w-info" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 11v6m0-10h.01" />
        </symbol>
      </svg>
      <aside className="wallet-sidebar" aria-label="Account navigation">
        <a
          className="wallet-brand"
          href="/dashboard"
          aria-label="Agrilinks Nigeria dashboard"
        >
          <span className="wallet-brand-mark">
            <svg className="w-icon">
              <use href="#w-shield" />
            </svg>
          </span>
          <span>
            Agrilinks Nigeria
            <span className="wallet-brand-caption">ROOTED IN POSSIBILITY</span>
          </span>
        </a>
        <button
          className="wallet-menu-toggle"
          type="button"
          aria-controls="wallet-navigation"
          aria-expanded="false"
          aria-label="Open navigation"
        >
          <span></span>
          <span></span>
        </button>
        <nav
          className="wallet-navigation"
          id="wallet-navigation"
          aria-label="Main navigation"
        >
          <span className="wallet-nav-label">YOUR WORKSPACE</span>
          <a href="/dashboard">
            <svg className="w-icon">
              <use href="#w-grid" />
            </svg>
            Overview
          </a>
          <a href="/marketplace">
            <svg className="w-icon">
              <use href="#w-market" />
            </svg>
            Marketplace
          </a>
          <a href="/orders">
            <svg className="w-icon">
              <use href="#w-bag" />
            </svg>
            My orders
          </a>
          <a href="/wallet" className="is-current" aria-current="page">
            <svg className="w-icon">
              <use href="#w-wallet" />
            </svg>
            My wallet<span className="wallet-nav-dot"></span>
          </a>
          <a href="/payments">
            <svg className="w-icon">
              <use href="#w-card" />
            </svg>
            Payments
          </a>
          <a href="/chat">
            <svg className="w-icon">
              <use href="#w-chat" />
            </svg>
            Messages
          </a>
          {truth(data?.User?.IsAdministrator) ? (
            <>
              <a href="/admin/finance">
                <svg className="w-icon">
                  <use href="#w-bank" />
                </svg>
                Admin workspace
              </a>
            </>
          ) : null}
          <a className="wallet-mobile-profile" href="/profile">
            <svg className="w-icon">
              <use href="#w-user" />
            </svg>
            My profile
          </a>
          <a className="wallet-mobile-profile" href="/logout">
            <svg className="w-icon">
              <use href="#w-logout" />
            </svg>
            Log out
          </a>
        </nav>
        <div className="wallet-sidebar-bottom">
          <div className="wallet-sidebar-note">
            <svg className="w-icon">
              <use href="#w-leaf" />
            </svg>
            <p>
              Big dreams.
              <br />
              <em>Solid roots.</em>
            </p>
            <span>
              A little care today.
              <br />A better harvest tomorrow.
            </span>
          </div>
          <a className="wallet-sidebar-profile" href="/profile">
            <span className="wallet-avatar" data-initials="">
              {truth(data?.User?.PhotoURL) ? (
                <>
                  <img src={text(data?.User?.PhotoURL)} alt="" />
                </>
              ) : (
                <>
                  <svg className="w-icon">
                    <use href="#w-user" />
                  </svg>
                </>
              )}
            </span>
            <span>
              <strong>{text(data?.User?.FullName)}</strong>
              <small>My account</small>
            </span>
            <svg className="w-icon">
              <use href="#w-chevron" />
            </svg>
          </a>
          <a className="wallet-logout" href="/logout">
            <svg className="w-icon">
              <use href="#w-logout" />
            </svg>
            Log out
          </a>
        </div>
      </aside>
      <main className="wallet-main" id="wallet-main">
        <header className="wallet-topbar">
          <p className="wallet-breadcrumb">
            My account<span>/</span>
            <strong>Wallet</strong>
          </p>
          <div className="wallet-topbar-actions">
            <time id="wallet-date"></time>
            <span className="wallet-topbar-divider"></span>
            <a
              className="wallet-icon-button"
              href="/notifications"
              aria-label="Notifications"
            >
              <svg className="w-icon">
                <use href="#w-bell" />
              </svg>
            </a>
            <a
              className="wallet-avatar wallet-topbar-avatar"
              href="/profile"
              aria-label="My profile"
              data-initials=""
            >
              {truth(data?.User?.PhotoURL) ? (
                <>
                  <img src={text(data?.User?.PhotoURL)} alt="" />
                </>
              ) : (
                <>
                  <svg className="w-icon">
                    <use href="#w-user" />
                  </svg>
                </>
              )}
            </a>
          </div>
        </header>
        <section className="wallet-intro" aria-labelledby="wallet-title">
          <div>
            <p className="wallet-eyebrow">
              <span></span> YOUR MONEY, AT A GLANCE
            </p>
            <h1 id="wallet-title">
              Your wallet.
              <br className="wallet-mobile-break" />
              <em>Room to grow.</em>
            </h1>
            <p className="wallet-intro-copy">
              Add money, pay for orders and manage transfers in one place.
            </p>
          </div>
          <span className="wallet-active">
            <span></span>Personal wallet
          </span>
        </section>
        {truth(data?.Error) ? (
          <>
            <div className="wallet-alert wallet-alert-error" role="alert">
              <svg className="w-icon">
                <use href="#w-info" />
              </svg>
              <p>{text(data?.Error)}</p>
            </div>
          </>
        ) : null}
        {truth(data?.Notice) ? (
          <>
            <div className="wallet-alert" role="status">
              <svg className="w-icon">
                <use href="#w-check" />
              </svg>
              <p>{text(data?.Notice)}</p>
            </div>
          </>
        ) : null}
        {truth(data?.Quote) ? (
          <>
            <section
              id="wallet-quote"
              className="wallet-move-panel"
              aria-labelledby="fee-review-title"
            >
              <div className="wallet-move-body">
                <h2 id="fee-review-title">
                  Review your{" "}
                  {truth(data?.Quote?.Kind === "withdrawal") ? (
                    <>withdrawal</>
                  ) : (
                    <>funding</>
                  )}
                </h2>
                <dl className="wallet-fee-breakdown">
                  <div>
                    <dt>Requested amount</dt>
                    <dd>₦{text(data?.Quote?.Amount)}</dd>
                  </div>
                  <div>
                    <dt>Agrilinks Nigeria service fee</dt>
                    <dd>₦{text(data?.Quote?.ServiceFee)}</dd>
                  </div>
                  <div>
                    <dt>Provider charge to you</dt>
                    <dd>₦{text(data?.Quote?.ProviderFee)}</dd>
                  </div>
                  {truth(data?.Quote?.Kind === "withdrawal") ? (
                    <>
                      <div>
                        <dt>Total wallet debit</dt>
                        <dd>₦{text(data?.Quote?.WalletDebit)}</dd>
                      </div>
                      <div>
                        <dt>Bank account receives</dt>
                        <dd>₦{text(data?.Quote?.BankCredit)}</dd>
                      </div>
                    </>
                  ) : (
                    <>
                      <div>
                        <dt>Total amount payable</dt>
                        <dd>₦{text(data?.Quote?.CheckoutTotal)}</dd>
                      </div>
                      <div>
                        <dt>Your wallet receives</dt>
                        <dd>₦{text(data?.Quote?.WalletCredit)}</dd>
                      </div>
                    </>
                  )}
                </dl>
                {truth(data?.Quote?.Kind === "withdrawal") ? (
                  <>
                    <p>
                      {text(data?.Quote?.BankName)} ·{" "}
                      {text(data?.Quote?.AccountNumber)} ·{" "}
                      {text(data?.Quote?.AccountName)}
                    </p>
                    <form
                      method="POST"
                      action="/wallet"
                      className="wallet-code-form"
                    >
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="send_code"
                      />
                      <input
                        type="hidden"
                        name="purpose"
                        defaultValue="withdrawal"
                      />
                      <input
                        type="hidden"
                        name="quote"
                        defaultValue={text(data?.Quote?.ID)}
                      />
                      <button>Send withdrawal code</button>
                    </form>
                  </>
                ) : null}
                {truth(data?.Quote?.Kind === "funding_preview") ? (
                  <>
                    <p>
                      Agrilinks Nigeria absorbs collection charges imposed on
                      its merchant account for this bank transfer. Your sending
                      bank may charge its own fee separately. Transfer the exact
                      confirmed total.
                    </p>
                  </>
                ) : null}
                {truth(data?.Quote?.Kind === "funding_allocation") ? (
                  <>
                    <p>
                      These funds have already arrived. Confirm this new wallet
                      credit and service fee to allocate them. No additional
                      payment will be collected.
                    </p>
                  </>
                ) : null}
                <p>
                  Review valid until{" "}
                  {text(formatDate(data?.Quote?.ExpiresAt, "15:04 MST"))}. Fees
                  are confirmed by Agrilinks Nigeria before processing.
                </p>
                <form method="POST" action="/wallet" className="wallet-form">
                  <input
                    type="hidden"
                    name="request_key"
                    defaultValue={text(data?.Quote?.ID)}
                  />
                  <input
                    type="hidden"
                    name="amount"
                    defaultValue={text(data?.Quote?.Amount)}
                  />
                  {truth(data?.Quote?.Kind === "withdrawal") ? (
                    <>
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="withdraw"
                      />
                      <label htmlFor="wallet-withdraw-code">
                        Verification code
                      </label>
                      <input
                        id="wallet-withdraw-code"
                        name="code"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="one-time-code"
                        required={true}
                      />
                      <label htmlFor="wallet-withdraw-pin">
                        Transaction PIN
                      </label>
                      <input
                        id="wallet-withdraw-pin"
                        name="transaction_pin"
                        type="password"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="off"
                        required={true}
                      />
                    </>
                  ) : truth(data?.Quote?.Kind === "funding_allocation") ? (
                    <>
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="allocate_funding"
                      />
                    </>
                  ) : truth(data?.Quote?.Kind === "funding_preview") ? (
                    <>
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="accept_funding"
                      />
                    </>
                  ) : (
                    <>
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="deposit"
                      />
                    </>
                  )}
                  <button className="wallet-submit" type="submit">
                    Confirm{" "}
                    {truth(data?.Quote?.Kind === "withdrawal") ? (
                      <>withdrawal</>
                    ) : truth(data?.Quote?.Kind === "funding_preview") ? (
                      <>bank funding instruction</>
                    ) : truth(data?.Quote?.Kind === "funding_allocation") ? (
                      <>allocation of existing funds</>
                    ) : (
                      <>and continue to payment</>
                    )}
                  </button>
                  <a href="/wallet">Change amount</a>
                </form>
              </div>
            </section>
          </>
        ) : null}
        {truth(data?.Escrows) ? (
          <>
            <details className="wallet-move-panel">
              <summary>
                <strong>Protected purchases and held seller funds</strong>
              </summary>
              <div className="wallet-move-body">
                {truth(data?.Escrows)
                  ? items(data?.Escrows).map((item1, index1) => (
                      <Fragment key={index1}>
                        <p>
                          <a
                            href={
                              "/orders/protection?order_id=" +
                              text(item1?.OrderID)
                            }
                          >
                            Order #{text(item1?.OrderID)} ·{" "}
                            {text(item1?.Products)}
                          </a>{" "}
                          — {text(item1?.Currency)}
                          {text(item1?.Amount)} · {text(item1?.State)}
                        </p>
                      </Fragment>
                    ))
                  : null}
                <p>
                  Held and disputed purchase funds are excluded from your
                  spendable wallet balance.
                </p>
              </div>
            </details>
          </>
        ) : null}
        <section className="wallet-overview" aria-label="Wallet overview">
          <div className="wallet-card-area">
            <div className="wallet-section-label">
              <h2>Your wallet</h2>
              <span>{text(data?.Wallet?.Currency)} account</span>
            </div>
            <article
              className="wallet-metal-card"
              id="wallet-metal-card"
              data-finish="gold"
              aria-label="Wallet balance and account details"
            >
              <div className="wallet-card-engraving" aria-hidden="true">
                <span></span>
                <span></span>
                <span></span>
                <span></span>
                <span></span>
              </div>
              <div className="wallet-card-top">
                <span className="wallet-card-brand">
                  <svg className="w-icon">
                    <use href="#w-shield" />
                  </svg>
                  Agrilinks Nigeria
                </span>
                <span className="wallet-card-finish" id="wallet-card-finish">
                  GOLD EDITION
                </span>
              </div>
              <div className="wallet-card-balance">
                <div className="wallet-balance-label">
                  Available balance
                  <button
                    type="button"
                    className="wallet-balance-toggle wallet-js-control"
                    id="wallet-balance-toggle"
                    aria-label="Hide balance"
                    aria-pressed="false"
                  >
                    <svg className="w-icon">
                      <use href="#w-eye" />
                    </svg>
                  </button>
                </div>
                <p
                  className="wallet-balance"
                  id="wallet-balance"
                  data-balance={text(data?.Wallet?.Balance)}
                  data-currency={text(data?.Wallet?.Currency)}
                >
                  <span className="wallet-currency">
                    {truth(data?.Wallet?.Currency === "NGN") ? (
                      <>₦</>
                    ) : (
                      <>{text(data?.Wallet?.Currency)}</>
                    )}
                  </span>
                  <span id="wallet-balance-value">
                    {text(data?.Wallet?.Balance)}
                  </span>
                </p>
                <span className="wallet-balance-caption">
                  Available to spend or transfer.
                </span>
              </div>
              <div className="wallet-card-bottom">
                <div>
                  <span className="wallet-card-caption">WALLET NUMBER</span>
                  <div className="wallet-number">
                    <span id="wallet-number">
                      {text(data?.Wallet?.WalletNumber)}
                    </span>
                    <button
                      type="button"
                      className="wallet-copy wallet-js-control"
                      data-copy-target="wallet-number"
                      aria-label="Copy wallet number"
                    >
                      <svg className="w-icon">
                        <use href="#w-copy" />
                      </svg>
                    </button>
                  </div>
                </div>
                <div className="wallet-card-owner">
                  <span className="wallet-card-caption">WALLET HOLDER</span>
                  <strong>{text(data?.User?.FullName)}</strong>
                </div>
                <svg
                  className="wallet-card-chip"
                  viewBox="0 0 44 34"
                  aria-hidden="true"
                >
                  <rect x="1" y="1" width="42" height="32" rx="7" />
                  <path d="M15 1v32M29 1v32M1 11h14m14 0h14M1 23h14m14 0h14M15 17h14" />
                </svg>
              </div>
            </article>
            <div className="wallet-finish-picker wallet-js-control">
              <p>
                Make it yours.<span>Choose your finish</span>
              </p>
              <div role="group" aria-label="Wallet card finish">
                <button
                  type="button"
                  data-wallet-finish="gold"
                  aria-pressed="true"
                >
                  <span className="wallet-swatch wallet-swatch-gold"></span>Gold
                </button>
                <button
                  type="button"
                  data-wallet-finish="silver"
                  aria-pressed="false"
                >
                  <span className="wallet-swatch wallet-swatch-silver"></span>
                  Silver
                </button>
                <button
                  type="button"
                  data-wallet-finish="bronze"
                  aria-pressed="false"
                >
                  <span className="wallet-swatch wallet-swatch-bronze"></span>
                  Bronze
                </button>
              </div>
            </div>
          </div>
          <div className="wallet-quick-area">
            <div className="wallet-section-label">
              <h2>Make your next move</h2>
              <svg className="w-icon">
                <use href="#w-up-right" />
              </svg>
            </div>
            <div className="wallet-quick-actions">
              <a
                className="wallet-quick-action wallet-action-gold"
                href="#wallet-funding"
              >
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-plus" />
                  </svg>
                </span>
                <span>
                  <strong>Add money</strong>
                  <small>Bank transfer or online payment</small>
                </span>
                <svg className="w-icon wallet-action-arrow">
                  <use href="#w-arrow" />
                </svg>
              </a>
              <a
                className="wallet-quick-action wallet-action-bronze"
                href="#wallet-transfer"
              >
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-send" />
                  </svg>
                </span>
                <span>
                  <strong>Send money</strong>
                  <small>From your wallet to theirs</small>
                </span>
                <svg className="w-icon wallet-action-arrow">
                  <use href="#w-arrow" />
                </svg>
              </a>
              <a
                className="wallet-quick-action wallet-action-silver"
                href="#wallet-withdraw"
              >
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-down" />
                  </svg>
                </span>
                <span>
                  <strong>Withdraw</strong>
                  <small>Move money to your bank account</small>
                </span>
                <svg className="w-icon wallet-action-arrow">
                  <use href="#w-arrow" />
                </svg>
              </a>
            </div>
            <p className="wallet-reserve-note">
              <svg className="w-icon">
                <use href="#w-shield" />
              </svg>
              <span>
                Pending transfers and withdrawals are held until completed or
                cancelled.
              </span>
            </p>
          </div>
        </section>
        <section
          className="wallet-funding"
          id="wallet-funding"
          aria-labelledby="wallet-funding-title"
        >
          <div className="wallet-funding-main">
            <div className="wallet-funding-heading">
              <span className="wallet-funding-icon">
                <svg className="w-icon">
                  <use href="#w-bank" />
                </svg>
              </span>
              <div>
                <p className="wallet-eyebrow">BANK TRANSFER</p>
                <h2 id="wallet-funding-title">Your personal funding account</h2>
              </div>
              <span
                className="wallet-funding-badge"
                data-status={
                  truth(
                    truth(data?.VirtualAccount) &&
                      truth(data?.VirtualAccount?.Ready),
                  )
                    ? "active"
                    : truth(data?.VirtualAccount)
                      ? text(data?.VirtualAccount?.Status)
                      : "new"
                }
              >
                {truth(
                  truth(data?.VirtualAccount) &&
                    truth(data?.VirtualAccount?.Ready),
                ) ? (
                  <>Ready to receive</>
                ) : truth(data?.VirtualAccount) ? (
                  <>
                    {truth(data?.VirtualAccount?.Status === "pending") ? (
                      <>Awaiting confirmation</>
                    ) : (
                      <>Setup incomplete</>
                    )}
                  </>
                ) : (
                  <>One-time setup</>
                )}
              </span>
            </div>
            {truth(
              truth(data?.VirtualAccount) && truth(data?.VirtualAccount?.Ready),
            ) ? (
              <>
                <p className="wallet-funding-lead">
                  Review and confirm your funding amount before transferring.
                  Your service fee is paid on top so your wallet receives the
                  full amount requested.
                </p>
                {truth(data?.FundingIntent) ? (
                  <>
                    <div className="wallet-alert">
                      <p>
                        Confirmed instruction: transfer{" "}
                        <strong>
                          ₦{text(data?.FundingIntent?.CheckoutTotal)}
                        </strong>{" "}
                        for a wallet credit of{" "}
                        <strong>
                          ₦{text(data?.FundingIntent?.WalletCredit)}
                        </strong>
                        . Agrilinks Nigeria fee: ₦
                        {text(data?.FundingIntent?.ServiceFee)}. Provider charge
                        to you: ₦{text(data?.FundingIntent?.ProviderFee)}. Valid
                        for 24 hours after confirmation.
                      </p>
                    </div>
                  </>
                ) : null}
                <form method="POST" action="/wallet" className="wallet-form">
                  <input
                    type="hidden"
                    name="action"
                    defaultValue="quote_funding"
                  />
                  <label htmlFor="funding-credit">
                    Amount to credit to your wallet (NGN)
                  </label>
                  <input
                    id="funding-credit"
                    type="number"
                    name="amount"
                    min="0.01"
                    step="0.01"
                    required={true}
                  />
                  <button className="wallet-submit">
                    Review bank funding total
                  </button>
                </form>
                <div className="wallet-funding-details">
                  <div>
                    <span className="wallet-detail-label">BANK NAME</span>
                    <strong>{text(data?.VirtualAccount?.BankName)}</strong>
                  </div>
                  <div className="wallet-funding-number-row">
                    <span className="wallet-detail-label">ACCOUNT NUMBER</span>
                    <div>
                      <strong id="wallet-funding-number">
                        {text(data?.VirtualAccount?.AccountNumber)}
                      </strong>
                      <button
                        type="button"
                        className="wallet-funding-copy wallet-js-control"
                        data-copy-target="wallet-funding-number"
                      >
                        <svg className="w-icon">
                          <use href="#w-copy" />
                        </svg>
                        Copy number
                      </button>
                    </div>
                  </div>
                  <div>
                    <span className="wallet-detail-label">ACCOUNT NAME</span>
                    <strong>{text(data?.VirtualAccount?.AccountName)}</strong>
                  </div>
                </div>
                <p className="wallet-funding-footnote">
                  <svg className="w-icon">
                    <use href="#w-check" />
                  </svg>
                  Account ready. Transfers are credited after payment
                  confirmation.
                </p>
                {truth(!truth(data?.ProviderAvailable)) ? (
                  <>
                    <p className="wallet-field-error">
                      Transfer confirmations are temporarily unavailable. Please
                      check again before sending money.
                    </p>
                  </>
                ) : null}
              </>
            ) : truth(!truth(data?.ProviderAvailable)) ? (
              <>
                <p className="wallet-funding-lead">
                  Funding account setup is temporarily unavailable. Please check
                  again later.
                </p>
              </>
            ) : truth(data?.Wallet?.Currency !== "NGN") ? (
              <>
                <p className="wallet-funding-lead">
                  Personal bank transfer accounts are available for NGN wallets.
                </p>
                <a className="wallet-inline-link" href="#wallet-deposit">
                  View online payment options
                  <svg className="w-icon">
                    <use href="#w-arrow" />
                  </svg>
                </a>
              </>
            ) : truth(
                truth(data?.VirtualAccount) &&
                  truth(data?.VirtualAccount?.Status === "pending"),
              ) ? (
              <>
                <p className="wallet-funding-lead">
                  Your account request is being confirmed. Your bank details
                  will appear here when the account is ready.
                </p>
                <div className="wallet-funding-progress">
                  <svg className="w-icon">
                    <use href="#w-clock" />
                  </svg>
                  <div>
                    <strong>Waiting for your bank details</strong>
                    <p>
                      You can leave this page and return later. We will keep
                      checking your existing request.
                    </p>
                  </div>
                </div>
                <form
                  method="POST"
                  action="/wallet"
                  className="wallet-funding-check"
                >
                  <input
                    type="hidden"
                    name="action"
                    defaultValue="check_funding"
                  />
                  <button type="submit" className="wallet-submit">
                    Check account status
                    <svg className="w-icon">
                      <use href="#w-arrow" />
                    </svg>
                  </button>
                </form>
                <p className="wallet-funding-footnote">
                  If the request stays pending, share setup reference{" "}
                  <strong>{text(data?.VirtualAccount?.TxRef)}</strong> with an
                  administrator. Never share your BVN or NIN in a message.
                </p>
              </>
            ) : truth(!truth(data?.FundingProfileReady)) ? (
              <>
                <p className="wallet-funding-lead">
                  Add your full name and a valid email address to your profile
                  before creating a funding account.
                </p>
                <a
                  className="wallet-funding-profile"
                  href="/profile#edit-profile-btn"
                >
                  Update your profile
                  <svg className="w-icon">
                    <use href="#w-arrow" />
                  </svg>
                </a>
              </>
            ) : (
              <>
                <p className="wallet-funding-lead">
                  {truth(data?.VirtualAccount) ? (
                    <>
                      Your account could not be created. Check your details and
                      try again.
                    </>
                  ) : (
                    <>
                      Set up a permanent NGN account to fund your wallet by bank
                      transfer. You only need to do this once.
                    </>
                  )}
                </p>
                <form
                  method="POST"
                  action="/wallet"
                  className="wallet-form wallet-identity-form"
                  data-virtual-account-form=""
                  autoComplete="off"
                >
                  <input
                    type="hidden"
                    name="action"
                    defaultValue="setup_funding"
                  />
                  <div className="wallet-identity-fields">
                    <div>
                      <label htmlFor="wallet-identity-kind">
                        Identity document
                      </label>
                      <Select id="wallet-identity-kind" name="identity_kind">
                        <option value="bvn">
                          Bank Verification Number (BVN)
                        </option>
                        <option value="nin">
                          National Identification Number (NIN)
                        </option>
                      </Select>
                    </div>
                    <div>
                      <label htmlFor="wallet-identity-number">
                        BVN or NIN number
                      </label>
                      <input
                        id="wallet-identity-number"
                        name="identity_number"
                        type="password"
                        inputMode="numeric"
                        pattern="[0-9]{11}"
                        minLength="11"
                        maxLength="11"
                        placeholder="11-digit number"
                        required={true}
                        aria-describedby="wallet-identity-help"
                        autoComplete="off"
                      />
                    </div>
                  </div>
                  <p id="wallet-identity-help" className="wallet-identity-help">
                    Use the identity details registered in your name. We do not
                    store your BVN or NIN.
                  </p>
                  <label
                    className="wallet-consent"
                    htmlFor="wallet-funding-consent"
                  >
                    <input
                      id="wallet-funding-consent"
                      type="checkbox"
                      name="consent"
                      defaultValue="yes"
                      required={true}
                    />
                    <span>
                      I agree to share my name, email, phone number and the BVN
                      or NIN entered above with Flutterwave to create my
                      permanent NGN funding account.
                    </span>
                  </label>
                  <button
                    type="submit"
                    className="wallet-submit"
                    data-va-submit=""
                  >
                    {truth(data?.VirtualAccount) ? (
                      <>Retry account setup</>
                    ) : (
                      <>Create funding account</>
                    )}
                    <svg className="w-icon">
                      <use href="#w-arrow" />
                    </svg>
                  </button>
                  <p
                    className="wallet-identity-help"
                    data-va-status=""
                    role="status"
                    aria-live="polite"
                  ></p>
                </form>
              </>
            )}
          </div>
          <aside
            className="wallet-funding-guide"
            aria-label="How bank funding works"
          >
            <p className="wallet-eyebrow">A SIMPLER WAY TO ADD MONEY</p>
            <h3>
              From your bank.
              <br /> Into your wallet.
            </h3>
            <ol>
              <li>
                <span>01</span>
                <div>
                  <strong>Set up your account</strong>
                  <p>Confirm your identity details once.</p>
                </div>
              </li>
              <li>
                <span>02</span>
                <div>
                  <strong>Make a bank transfer</strong>
                  <p>Check the bank and account name before sending.</p>
                </div>
              </li>
              <li>
                <span>03</span>
                <div>
                  <strong>Follow your balance</strong>
                  <p>Your balance updates after the transfer is confirmed.</p>
                </div>
              </li>
            </ol>
            <a href="#wallet-deposit" className="wallet-funding-alternative">
              Prefer online payment?
              <span>
                View other funding options
                <svg className="w-icon">
                  <use href="#w-arrow" />
                </svg>
              </span>
            </a>
          </aside>
        </section>
        <div
          className="wallet-alert wallet-update-notice"
          id="wallet-update-notice"
          role="status"
          hidden={true}
        >
          <svg className="w-icon">
            <use href="#w-check" />
          </svg>
          <p>
            Your wallet has been updated. Refresh when you are ready to see the
            latest balance and activity.
          </p>
          <a href="/wallet">Refresh wallet</a>
        </div>
        <div className="wallet-content-grid">
          <section
            className="wallet-activity wallet-surface"
            aria-labelledby="wallet-activity-title"
          >
            <div className="wallet-panel-heading">
              <div>
                <p className="wallet-eyebrow">RECENT ACTIVITY</p>
                <h2 id="wallet-activity-title">Transaction history</h2>
              </div>
              <span className="wallet-heading-icon">
                <svg className="w-icon">
                  <use href="#w-clock" />
                </svg>
              </span>
            </div>
            <div
              className="wallet-history-filters wallet-js-control"
              role="group"
              aria-label="Transaction history category"
            >
              <button
                type="button"
                data-history-filter="balance"
                aria-pressed="true"
              >
                Balance history
              </button>
              <button
                type="button"
                data-history-filter="funding"
                aria-pressed="false"
              >
                Funding<span>{text(len(data?.Deposits))}</span>
              </button>
              <button
                type="button"
                data-history-filter="transfers"
                aria-pressed="false"
              >
                Transfers<span>{text(len(data?.Transfers))}</span>
              </button>
              <button
                type="button"
                data-history-filter="withdrawals"
                aria-pressed="false"
              >
                Withdrawals<span>{text(len(data?.Withdrawals))}</span>
              </button>
            </div>
            <section
              className="wallet-history-section"
              data-history="balance"
              aria-label="Balance history"
            >
              <h3 className="wallet-history-fallback-title">Balance history</h3>
              {truth(data?.History) ? (
                items(data?.History).map((item2, index2) => (
                  <Fragment key={index2}>
                    <article className="wallet-transaction">
                      <span className="wallet-transaction-icon">
                        <svg className="w-icon">
                          <use href="#w-wallet" />
                        </svg>
                      </span>
                      <div className="wallet-transaction-description">
                        <h4>{text(humanStatus(item2?.Type))}</h4>
                        <p>{text(item2?.Reference)}</p>
                      </div>
                      <div className="wallet-transaction-value">
                        <strong
                          className="wallet-history-money"
                          data-money={text(item2?.Amount)}
                          data-money-currency={text(data?.Wallet?.Currency)}
                        >
                          {text(data?.Wallet?.Currency)}
                          {text(item2?.Amount)}
                        </strong>
                        <time
                          dateTime={text(
                            formatDate(
                              item2?.CreatedAt,
                              "2006-01-02T15:04:05Z07:00",
                            ),
                          )}
                        >
                          {text(
                            formatDate(item2?.CreatedAt, "02 Jan 2006 · 15:04"),
                          )}
                        </time>
                      </div>
                    </article>
                  </Fragment>
                ))
              ) : (
                <>
                  <div className="wallet-empty">
                    <span className="wallet-empty-icon">
                      <svg className="w-icon">
                        <use href="#w-leaf" />
                      </svg>
                    </span>
                    <h3>A fresh start. A world of possibility.</h3>
                    <p>
                      Your balance activity will appear here.
                      <br />
                      Your first move is the beginning of something good.
                    </p>
                    <a href="#wallet-deposit">
                      Add your first funds
                      <svg className="w-icon">
                        <use href="#w-arrow" />
                      </svg>
                    </a>
                  </div>
                </>
              )}
              <p className="wallet-history-footnote">
                Your latest 200 balance changes. All in one place.
              </p>
            </section>
            <section
              className="wallet-history-section"
              data-history="funding"
              aria-label="Funding history"
            >
              <h3 className="wallet-history-fallback-title">Funding history</h3>
              {truth(data?.Deposits) ? (
                items(data?.Deposits).map((item3, index3) => (
                  <Fragment key={index3}>
                    <article className="wallet-transaction">
                      <span className="wallet-transaction-icon is-gold">
                        <svg className="w-icon">
                          <use href="#w-plus" />
                        </svg>
                      </span>
                      <div className="wallet-transaction-description">
                        <h4>Wallet funding</h4>
                        <p>{text(item3?.Reference)}</p>
                        {withValue(
                          data?.FeeHistory?.[item3?.Reference],
                          (item4) => (
                            <>
                              <p>
                                Wallet credit: ₦{text(item4?.WalletCredit)} ·
                                Agrilinks Nigeria fee: ₦
                                {text(item4?.ServiceFee)} · Provider charge: ₦
                                {text(item4?.ProviderFee)} ·{" "}
                                {text(item4?.State)}
                              </p>
                              {truth(item4?.State === "review") ? (
                                <>
                                  <p>
                                    Funds are held for review because they do
                                    not match a confirmed funding instruction.
                                    No service fee was collected.
                                  </p>
                                  <form method="POST" action="/wallet">
                                    <input
                                      type="hidden"
                                      name="action"
                                      defaultValue="quote_allocation"
                                    />
                                    <input
                                      type="hidden"
                                      name="reference"
                                      defaultValue={text(item3?.Reference)}
                                    />
                                    <button className="wallet-inline-link">
                                      Review allocation of these funds
                                    </button>
                                  </form>
                                </>
                              ) : null}
                            </>
                          ),
                        )}
                        {truth(
                          truth(item3?.Status === "pending") &&
                            truth(item3?.CheckoutURL),
                        ) ? (
                          <>
                            <a
                              className="wallet-inline-link"
                              href={text(item3?.CheckoutURL)}
                            >
                              Continue this payment
                            </a>
                          </>
                        ) : null}
                        <time
                          dateTime={text(
                            formatDate(
                              item3?.CreatedAt,
                              "2006-01-02T15:04:05Z07:00",
                            ),
                          )}
                        >
                          {text(
                            formatDate(item3?.CreatedAt, "02 Jan 2006 · 15:04"),
                          )}
                        </time>
                      </div>
                      <div className="wallet-transaction-value">
                        <strong
                          data-money={text(item3?.Amount)}
                          data-money-currency={text(item3?.Currency)}
                        >
                          {text(item3?.Currency)}
                          {text(item3?.Amount)}
                        </strong>
                        <span
                          className="wallet-status"
                          data-status={text(item3?.Status)}
                        >
                          {text(humanPaymentStatus(item3?.Status))}
                        </span>
                      </div>
                    </article>
                  </Fragment>
                ))
              ) : (
                <>
                  <div className="wallet-empty">
                    <span className="wallet-empty-icon">
                      <svg className="w-icon">
                        <use href="#w-plus" />
                      </svg>
                    </span>
                    <h3>Make room for what's next.</h3>
                    <p>No funding requests yet. Add money to get started.</p>
                    <a href="#wallet-deposit">
                      Add money
                      <svg className="w-icon">
                        <use href="#w-arrow" />
                      </svg>
                    </a>
                  </div>
                </>
              )}
            </section>
            <section
              className="wallet-history-section"
              data-history="transfers"
              aria-label="Transfer history"
            >
              <h3 className="wallet-history-fallback-title">Transfers</h3>
              {truth(data?.Transfers) ? (
                items(data?.Transfers).map((item5, index5) => (
                  <Fragment key={index5}>
                    <article className="wallet-transaction">
                      <span className="wallet-transaction-icon is-bronze">
                        <svg className="w-icon">
                          <use href="#w-send" />
                        </svg>
                      </span>
                      <div className="wallet-transaction-description">
                        <h4>
                          {text(item5?.SenderName)}
                          <span aria-label="to">→</span>
                          {text(item5?.RecipientName)}
                        </h4>
                        <p>
                          {truth(item5?.Note) ? (
                            <>{text(item5?.Note)}</>
                          ) : (
                            <>{text(item5?.Reference)}</>
                          )}
                        </p>
                        <time
                          dateTime={text(
                            formatDate(
                              item5?.CreatedAt,
                              "2006-01-02T15:04:05Z07:00",
                            ),
                          )}
                        >
                          {text(
                            formatDate(item5?.CreatedAt, "02 Jan 2006 · 15:04"),
                          )}
                        </time>
                        {truth(
                          truth(item5?.Status === "pending") &&
                            truth(item5?.SenderID === data?.User?.ID),
                        ) ? (
                          <>
                            <form
                              method="POST"
                              action="/wallet"
                              className="wallet-cancel-form"
                            >
                              <input
                                type="hidden"
                                name="action"
                                defaultValue="cancel_transfer"
                              />
                              <input
                                type="hidden"
                                name="id"
                                defaultValue={text(item5?.ID)}
                              />
                              <button type="submit">
                                Cancel and return money
                              </button>
                            </form>
                          </>
                        ) : null}
                      </div>
                      <div className="wallet-transaction-value">
                        <strong
                          data-money={text(item5?.Amount)}
                          data-money-currency={text(item5?.Currency)}
                        >
                          {text(item5?.Currency)}
                          {text(item5?.Amount)}
                        </strong>
                        <span
                          className="wallet-status"
                          data-status={text(item5?.Status)}
                        >
                          {text(humanStatus(item5?.Status))}
                        </span>
                      </div>
                    </article>
                  </Fragment>
                ))
              ) : (
                <>
                  <div className="wallet-empty">
                    <span className="wallet-empty-icon">
                      <svg className="w-icon">
                        <use href="#w-send" />
                      </svg>
                    </span>
                    <h3>Good things are better shared.</h3>
                    <p>
                      Your incoming and outgoing transfers will appear here.
                    </p>
                    <a href="#wallet-transfer">
                      Send money
                      <svg className="w-icon">
                        <use href="#w-arrow" />
                      </svg>
                    </a>
                  </div>
                </>
              )}
            </section>
            <section
              className="wallet-history-section"
              data-history="withdrawals"
              aria-label="Withdrawal history"
            >
              <h3 className="wallet-history-fallback-title">Withdrawals</h3>
              {truth(data?.Withdrawals) ? (
                items(data?.Withdrawals).map((item6, index6) => (
                  <Fragment key={index6}>
                    <article className="wallet-transaction">
                      <span className="wallet-transaction-icon is-silver">
                        <svg className="w-icon">
                          <use href="#w-down" />
                        </svg>
                      </span>
                      <div className="wallet-transaction-description">
                        <h4>Withdrawal to {text(item6?.BankName)}</h4>
                        <p>{text(item6?.MaskedAccountNumber)}</p>
                        {withValue(
                          data?.FeeHistory?.[item6?.Reference],
                          (item7) => (
                            <>
                              <p>
                                Total debit: ₦{text(item7?.WalletDebit)} ·
                                Agrilinks Nigeria fee: ₦
                                {text(item7?.ServiceFee)} · Provider charge: ₦
                                {text(item7?.ProviderFee)} ·{" "}
                                {text(item7?.State)}
                              </p>
                            </>
                          ),
                        )}
                        <time
                          dateTime={text(
                            formatDate(
                              item6?.CreatedAt,
                              "2006-01-02T15:04:05Z07:00",
                            ),
                          )}
                        >
                          {text(
                            formatDate(item6?.CreatedAt, "02 Jan 2006 · 15:04"),
                          )}
                        </time>
                        {truth(item6?.Status === "pending") ? (
                          <>
                            <form
                              method="POST"
                              action="/wallet"
                              className="wallet-cancel-form"
                            >
                              <input
                                type="hidden"
                                name="action"
                                defaultValue="cancel_withdrawal"
                              />
                              <input
                                type="hidden"
                                name="id"
                                defaultValue={text(item6?.ID)}
                              />
                              <button type="submit">
                                Cancel and return money
                              </button>
                            </form>
                          </>
                        ) : null}
                      </div>
                      <div className="wallet-transaction-value">
                        <strong
                          data-money={text(item6?.Amount)}
                          data-money-currency={text(item6?.Currency)}
                        >
                          {text(item6?.Currency)}
                          {text(item6?.Amount)}
                        </strong>
                        <span
                          className="wallet-status"
                          data-status={text(item6?.Status)}
                        >
                          {text(humanStatus(item6?.Status))}
                        </span>
                      </div>
                    </article>
                  </Fragment>
                ))
              ) : (
                <>
                  <div className="wallet-empty">
                    <span className="wallet-empty-icon">
                      <svg className="w-icon">
                        <use href="#w-bank" />
                      </svg>
                    </span>
                    <h3>Your money. Your next destination.</h3>
                    <p>No withdrawals yet. Your requests will appear here.</p>
                    <a href="#wallet-withdraw">
                      Withdraw to your bank
                      <svg className="w-icon">
                        <use href="#w-arrow" />
                      </svg>
                    </a>
                  </div>
                </>
              )}
            </section>
          </section>
          <aside
            className="wallet-companion"
            aria-label="Bank details and marketplace"
          >
            <section
              className="wallet-bank-card"
              aria-labelledby="wallet-bank-title"
            >
              <div className="wallet-bank-card-top">
                <span className="wallet-bank-icon">
                  <svg className="w-icon">
                    <use href="#w-bank" />
                  </svg>
                </span>
                <span className="wallet-eyebrow">WITHDRAWALS</span>
              </div>
              <h2 id="wallet-bank-title">Your withdrawal account</h2>
              {truth(
                truth(data?.Profile) && truth(data?.Profile?.AccountNumber),
              ) ? (
                <>
                  <p className="wallet-bank-copy">
                    Your saved account for withdrawals.
                  </p>
                  <span className="wallet-bank-name">
                    {text(data?.Profile?.BankName)}
                  </span>
                  <div className="wallet-bank-number">
                    <strong>{text(data?.Profile?.AccountNumber)}</strong>
                  </div>
                  <p className="wallet-bank-holder">
                    {text(data?.Profile?.AccountName)}
                  </p>
                  <a className="wallet-bank-link" href="#wallet-bank-details">
                    Manage bank account
                    <svg className="w-icon">
                      <use href="#w-arrow" />
                    </svg>
                  </a>
                </>
              ) : (
                <>
                  <p className="wallet-bank-copy">
                    Connect your bank account to receive withdrawals from your
                    wallet.
                  </p>
                  <a className="wallet-bank-link" href="#wallet-bank-details">
                    Add withdrawal account
                    <svg className="w-icon">
                      <use href="#w-arrow" />
                    </svg>
                  </a>
                </>
              )}
            </section>
            <section className="wallet-growth-card">
              <svg
                className="wallet-growth-art"
                viewBox="0 0 160 170"
                aria-hidden="true"
              >
                <circle cx="84" cy="74" r="55" />
                <circle cx="84" cy="74" r="43" />
                <path d="M80 143V55m0 56C42 112 35 93 32 75c30 0 48 14 48 36Zm0-23c38 0 48-21 49-43-31 3-49 19-49 43Zm0-15c-23 0-31-18-30-36 19 3 30 15 30 36ZM58 145h46m-37 8h28" />
              </svg>
              <p className="wallet-eyebrow">ROOTED IN POSSIBILITY</p>
              <h2>
                Small seeds.
                <br />
                <em>Extraordinary futures.</em>
              </h2>
              <p>
                Put your wallet to work. Discover what your next harvest could
                look like.
              </p>
              <a href="/marketplace">
                Explore marketplace
                <svg className="w-icon">
                  <use href="#w-up-right" />
                </svg>
              </a>
            </section>
          </aside>
        </div>
        <section className="wallet-moves" aria-labelledby="wallet-moves-title">
          <div className="wallet-moves-heading">
            <div>
              <p className="wallet-eyebrow">LET'S MAKE IT HAPPEN</p>
              <h2 id="wallet-moves-title">Made for your next move.</h2>
            </div>
            <p>Simple steps. More possibilities.</p>
          </div>
          {truth(!truth(data?.HasTransactionPIN)) ? (
            <>
              <details
                className="wallet-move-panel"
                id="wallet-pin-setup"
                open={true}
              >
                <summary>
                  <span>
                    <strong>Secure your wallet</strong>
                    <small>Create your transaction PIN</small>
                  </span>
                </summary>
                <div className="wallet-move-body">
                  <p>
                    Create a six-digit PIN before making payments, transfers, or
                    withdrawals. Keep it safe and never reveal it to anyone.
                    Agrilinks Nigeria staff will never ask for it.
                  </p>
                  <form method="POST" action="/wallet" className="wallet-form">
                    <input
                      type="hidden"
                      name="action"
                      defaultValue="setup_pin"
                    />
                    <label htmlFor="wallet-current-password">
                      Current account password
                    </label>
                    <input
                      id="wallet-current-password"
                      name="current_password"
                      type="password"
                      autoComplete="current-password"
                      required={true}
                    />
                    <label htmlFor="wallet-new-pin">
                      Create transaction PIN
                    </label>
                    <input
                      id="wallet-new-pin"
                      name="transaction_pin"
                      type="password"
                      inputMode="numeric"
                      pattern="[0-9]{6}"
                      minLength="6"
                      maxLength="6"
                      autoComplete="new-password"
                      required={true}
                    />
                    <label htmlFor="wallet-confirm-pin">
                      Confirm transaction PIN
                    </label>
                    <input
                      id="wallet-confirm-pin"
                      name="transaction_pin_confirmation"
                      type="password"
                      inputMode="numeric"
                      pattern="[0-9]{6}"
                      minLength="6"
                      maxLength="6"
                      autoComplete="new-password"
                      required={true}
                    />
                    <button type="submit" className="wallet-submit">
                      Create transaction PIN
                    </button>
                  </form>
                </div>
              </details>
            </>
          ) : null}
          <div className="wallet-move-grid">
            <details
              className="wallet-move-panel wallet-move-gold"
              id="wallet-deposit"
            >
              <summary>
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-plus" />
                  </svg>
                </span>
                <span>
                  <strong>Online top-up</strong>
                  <small>More ways to add money</small>
                </span>
                <svg className="w-icon wallet-summary-chevron">
                  <use href="#w-chevron" />
                </svg>
              </summary>
              <div className="wallet-move-body">
                {truth(data?.ProviderAvailable) ? (
                  <>
                    <p>
                      Choose an amount, then continue to secure payment. Your
                      balance updates after confirmation.
                    </p>
                    <form
                      method="POST"
                      action="/wallet"
                      className="wallet-form"
                    >
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="quote_deposit"
                      />
                      <input
                        type="hidden"
                        name="request_key"
                        defaultValue={
                          truth(
                            truth(data?.FailedAction === "deposit") &&
                              truth(data?.RequestKey),
                          )
                            ? text(data?.RequestKey)
                            : text(requestKey())
                        }
                      />
                      <label htmlFor="wallet-payment-method">
                        Payment method
                      </label>
                      <Select id="wallet-payment-method" name="payment_method">
                        <option value="banktransfer">Bank transfer</option>
                        <option value="card">Card</option>
                      </Select>
                      <label htmlFor="wallet-deposit-amount">
                        Amount in {text(data?.Wallet?.Currency)}
                      </label>
                      <div className="wallet-amount-input">
                        <span>
                          {truth(data?.Wallet?.Currency === "NGN") ? (
                            <>₦</>
                          ) : (
                            <>{text(data?.Wallet?.Currency)}</>
                          )}
                        </span>
                        <input
                          id="wallet-deposit-amount"
                          name="amount"
                          type="number"
                          min="0.01"
                          step="0.01"
                          placeholder="0.00"
                          required={true}
                        />
                      </div>
                      <div
                        className="wallet-amount-presets wallet-js-control"
                        role="group"
                        aria-label="Suggested deposit amounts"
                      >
                        <button type="button" data-deposit-amount="1000">
                          1,000
                        </button>
                        <button type="button" data-deposit-amount="5000">
                          5,000
                        </button>
                        <button type="button" data-deposit-amount="10000">
                          10,000
                        </button>
                      </div>
                      <button type="submit" className="wallet-submit">
                        Review fees
                        <svg className="w-icon">
                          <use href="#w-arrow" />
                        </svg>
                      </button>
                    </form>
                  </>
                ) : (
                  <>
                    <p>
                      Bank payments are not set up yet. Your saved balance and
                      history are available above.
                    </p>
                  </>
                )}
              </div>
            </details>
            <details
              className="wallet-move-panel wallet-move-bronze"
              id="wallet-transfer"
            >
              <summary>
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-send" />
                  </svg>
                </span>
                <span>
                  <strong>Send money</strong>
                  <small>Good things, shared</small>
                </span>
                <svg className="w-icon wallet-summary-chevron">
                  <use href="#w-chevron" />
                </svg>
              </summary>
              <div className="wallet-move-body">
                <p>
                  Send to another Agrilinks Nigeria wallet. Request a
                  verification code first. Transfers are reviewed by an
                  administrator before completion.
                </p>
                <form
                  method="POST"
                  action="/wallet"
                  className="wallet-code-form"
                >
                  <input type="hidden" name="action" defaultValue="send_code" />
                  <input type="hidden" name="purpose" defaultValue="transfer" />
                  <button type="submit">
                    <svg className="w-icon">
                      <use href="#w-shield" />
                    </svg>
                    Send transfer code
                  </button>
                </form>
                <form method="POST" action="/wallet" className="wallet-form">
                  <input type="hidden" name="action" defaultValue="transfer" />
                  <input
                    type="hidden"
                    name="request_key"
                    defaultValue={
                      truth(
                        truth(data?.FailedAction === "transfer") &&
                          truth(data?.RequestKey),
                      )
                        ? text(data?.RequestKey)
                        : text(requestKey())
                    }
                  />
                  <label htmlFor="wallet-recipient">
                    Recipient wallet number
                  </label>
                  <input
                    id="wallet-recipient"
                    name="recipient"
                    placeholder="Enter their wallet number"
                    required={true}
                  />
                  <label htmlFor="wallet-transfer-amount">
                    Amount in {text(data?.Wallet?.Currency)}
                  </label>
                  <input
                    id="wallet-transfer-amount"
                    name="amount"
                    type="number"
                    min="0.01"
                    step="0.01"
                    placeholder="0.00"
                    required={true}
                  />
                  <label htmlFor="wallet-transfer-note">
                    Note <span>(optional)</span>
                  </label>
                  <input
                    id="wallet-transfer-note"
                    name="note"
                    maxLength="1000"
                    placeholder="What's it for?"
                  />
                  <label htmlFor="wallet-transfer-code">
                    Verification code
                  </label>
                  <input
                    id="wallet-transfer-code"
                    name="code"
                    inputMode="numeric"
                    pattern="[0-9]{6}"
                    maxLength="6"
                    autoComplete="one-time-code"
                    placeholder="6-digit code"
                    required={true}
                  />
                  <label htmlFor="wallet-transfer-pin">Transaction PIN</label>
                  <input
                    id="wallet-transfer-pin"
                    name="transaction_pin"
                    type="password"
                    inputMode="numeric"
                    pattern="[0-9]{6}"
                    maxLength="6"
                    autoComplete="off"
                    required={true}
                  />
                  <button type="submit" className="wallet-submit">
                    Request transfer
                    <svg className="w-icon">
                      <use href="#w-arrow" />
                    </svg>
                  </button>
                </form>
              </div>
            </details>
            <details
              className="wallet-move-panel wallet-move-silver"
              id="wallet-withdraw"
            >
              <summary>
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-down" />
                  </svg>
                </span>
                <span>
                  <strong>Withdraw money</strong>
                  <small>On to your next chapter</small>
                </span>
                <svg className="w-icon wallet-summary-chevron">
                  <use href="#w-chevron" />
                </svg>
              </summary>
              <div className="wallet-move-body">
                {truth(data?.ProviderAvailable) ? (
                  <>
                    <p>
                      Withdraw to your saved bank account. Request a code first.
                      Withdrawals to your verified own bank account are
                      submitted immediately after security checks.
                    </p>
                    {truth(!truth(data?.Profile?.AccountNumber)) ? (
                      <>
                        <a
                          className="wallet-inline-link"
                          href="#wallet-bank-details"
                        >
                          Set up your bank account first
                          <svg className="w-icon">
                            <use href="#w-arrow" />
                          </svg>
                        </a>
                      </>
                    ) : null}
                    <form
                      method="POST"
                      action="/wallet"
                      className="wallet-code-form"
                    >
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="send_code"
                      />
                      <input
                        type="hidden"
                        name="purpose"
                        defaultValue="withdrawal"
                      />
                      <button type="submit">
                        <svg className="w-icon">
                          <use href="#w-shield" />
                        </svg>
                        Send withdrawal code
                      </button>
                    </form>
                    <form
                      method="POST"
                      action="/wallet"
                      className="wallet-form"
                    >
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="quote_withdrawal"
                      />
                      <input
                        type="hidden"
                        name="request_key"
                        defaultValue={
                          truth(
                            truth(data?.FailedAction === "withdraw") &&
                              truth(data?.RequestKey),
                          )
                            ? text(data?.RequestKey)
                            : text(requestKey())
                        }
                      />
                      <label htmlFor="wallet-withdraw-amount">
                        Amount in whole naira
                      </label>
                      <input
                        id="wallet-withdraw-amount"
                        name="amount"
                        type="number"
                        min="1"
                        step="1"
                        placeholder="0"
                        required={true}
                      />
                      <button type="submit" className="wallet-submit">
                        Review withdrawal fees
                        <svg className="w-icon">
                          <use href="#w-arrow" />
                        </svg>
                      </button>
                    </form>
                  </>
                ) : (
                  <>
                    <p>
                      Bank withdrawals will be available when bank payments are
                      set up. Your saved balance and withdrawal history are
                      available above.
                    </p>
                  </>
                )}
              </div>
            </details>
            <details
              className="wallet-move-panel wallet-move-silver"
              id="wallet-bank-details"
            >
              <summary>
                <span className="wallet-action-icon">
                  <svg className="w-icon">
                    <use href="#w-bank" />
                  </svg>
                </span>
                <span>
                  <strong>Your bank account</strong>
                  <small>A place to land</small>
                </span>
                <svg className="w-icon wallet-summary-chevron">
                  <use href="#w-chevron" />
                </svg>
              </summary>
              <div className="wallet-move-body">
                {truth(data?.ProviderAvailable) ? (
                  <>
                    <p>
                      Choose your bank and enter your account number. We'll
                      check the account details before saving.
                    </p>
                    {truth(data?.Profile?.AccountNumber) ? (
                      <>
                        <div className="wallet-saved-bank">
                          <strong>{text(data?.Profile?.BankName)}</strong>
                          <span>
                            {text(data?.Profile?.AccountName)} ·{" "}
                            {text(data?.Profile?.AccountNumber)}
                          </span>
                        </div>
                      </>
                    ) : null}
                    <form
                      method="POST"
                      action="/wallet"
                      className="wallet-form"
                    >
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="save_bank"
                      />
                      <label htmlFor="wallet-bank">Bank</label>
                      <Select
                        name="bank_code"
                        id="wallet-bank"
                        required={true}
                        aria-describedby="bank-error"
                      >
                        <option value="">Loading banks…</option>
                      </Select>
                      <label htmlFor="wallet-account-number">
                        Account number
                      </label>
                      <input
                        id="wallet-account-number"
                        name="account_number"
                        inputMode="numeric"
                        pattern="[0-9]{10}"
                        maxLength="10"
                        placeholder="10-digit account number"
                        required={true}
                      />
                      <p
                        id="bank-error"
                        className="wallet-field-error"
                        role="alert"
                      ></p>
                      <label htmlFor="wallet-bank-pin">Transaction PIN</label>
                      <input
                        id="wallet-bank-pin"
                        name="transaction_pin"
                        type="password"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        maxLength="6"
                        autoComplete="off"
                        required={true}
                      />
                      <button
                        type="submit"
                        className="wallet-submit"
                        id="wallet-save-bank"
                      >
                        Check and save account
                        <svg className="w-icon">
                          <use href="#w-check" />
                        </svg>
                      </button>
                    </form>
                    <noscript>
                      <p className="wallet-field-error">
                        Please enable JavaScript to load the bank list.
                      </p>
                    </noscript>
                  </>
                ) : (
                  <>
                    <p>
                      Bank account setup will be available when bank payments
                      are configured.
                    </p>
                  </>
                )}
              </div>
            </details>
          </div>
        </section>
        <footer className="wallet-footer">
          <span>
            <svg className="w-icon">
              <use href="#w-shield" />
            </svg>
            Agrilinks Nigeria. Growing with you.
          </span>
          <span>Bronze roots. Silver moments. Golden possibilities.</span>
        </footer>
      </main>
      <div
        className="wallet-toast"
        id="wallet-toast"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      ></div>
    </>
  );
}
