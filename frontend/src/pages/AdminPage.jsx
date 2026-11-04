import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text, humanStatus, len } from "../lib/view.js";
import mountAdmin from "../behaviors/admin.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function AdminPage({ data }) {
  usePageBehaviors([mountAdmin], data);
  return (
    <>
      <a className="skip-link" href="#admin-content">
        Skip to content
      </a>
      <svg
        className="admin-icon-definitions"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <symbol id="a-shield" viewBox="0 0 24 24">
          <path d="M12 3 4 6v6c0 4 3.5 7 8 9 4.5-2 8-5 8-9V6Z" />
          <path d="M12 16V8m0 5c-4 0-5-3-5-3 4 0 5 3 5 3Zm0-2c4 0 5-3 5-3-4 0-5 3-5 3Z" />
        </symbol>
        <symbol id="a-overview" viewBox="0 0 24 24">
          <rect x="3" y="3" width="7" height="7" rx="1.5" />
          <rect x="14" y="3" width="7" height="7" rx="1.5" />
          <rect x="3" y="14" width="7" height="7" rx="1.5" />
          <rect x="14" y="14" width="7" height="7" rx="1.5" />
        </symbol>
        <symbol id="a-users" viewBox="0 0 24 24">
          <circle cx="9" cy="8" r="3" />
          <path d="M3 21v-3a6 6 0 0 1 12 0v3M16 5a3 3 0 0 1 0 6m3 10v-3a6 6 0 0 0-3-5" />
        </symbol>
        <symbol id="a-marketplace" viewBox="0 0 24 24">
          <path d="m3 10 2-6h14l2 6M5 13v7h14v-7M3 10c0 4 4 4 4 0 0 4 5 4 5 0 0 4 5 4 5 0 0 4 4 4 4 0M10 20v-5h4v5" />
        </symbol>
        <symbol id="a-orders" viewBox="0 0 24 24">
          <path d="M5 7h14l1 14H4L5 7Zm3 0V6a4 4 0 0 1 8 0v1" />
        </symbol>
        <symbol id="a-finance" viewBox="0 0 24 24">
          <path d="M20 8V5H5a2 2 0 0 0 0 4h15v11H5a2 2 0 0 1-2-2V7m17 5h-6v5h6" />
          <path d="M16 14.5h.01" />
        </symbol>
        <symbol id="a-support" viewBox="0 0 24 24">
          <path d="M4 14v-3a8 8 0 0 1 16 0v3M4 12H2v6h4v-6H4Zm16 0h2v6h-4v-6h2ZM20 18c0 3-4 3-6 3h-2" />
        </symbol>
        <symbol id="a-reports" viewBox="0 0 24 24">
          <path d="M4 3v17h17M8 16v-5m5 5V7m5 9V4" />
        </symbol>
        <symbol id="a-audit" viewBox="0 0 24 24">
          <path d="M8 3H4v18h16V3h-4M8 3V1h8v4H8V3Zm0 8h8m-8 4h5" />
        </symbol>
        <symbol id="a-staff" viewBox="0 0 24 24">
          <circle cx="8" cy="7" r="3" />
          <path d="M2 21v-3a6 6 0 0 1 11-3m4-3 5 2v4c0 2-2 4-5 5-3-1-5-3-5-5v-4l5-2Zm-2 5 2 2 3-3" />
        </symbol>
        <symbol id="a-settings" viewBox="0 0 24 24">
          <path d="M12 3 10 5H7L5 8v3l-2 2 2 2v3l3 2h3l2 2 2-2h3l2-3v-3l2-2-2-2V7l-3-2h-3L12 3Z" />
          <circle cx="12" cy="12" r="3" />
        </symbol>
        <symbol id="a-leaf" viewBox="0 0 24 24">
          <path d="M5 19C-1 9 10 3 21 3c0 11-6 22-16 16ZM4 21 16 9" />
        </symbol>
        <symbol id="a-arrow" viewBox="0 0 24 24">
          <path d="M5 12h14m-5-5 5 5-5 5" />
        </symbol>
        <symbol id="a-up-right" viewBox="0 0 24 24">
          <path d="M6 18 18 6M6 6h12v12" />
        </symbol>
        <symbol id="a-search" viewBox="0 0 24 24">
          <circle cx="10" cy="10" r="6" />
          <path d="m15 15 6 6" />
        </symbol>
        <symbol id="a-refresh" viewBox="0 0 24 24">
          <path d="M20 7a9 9 0 1 0 1 7M20 3v5h-5" />
        </symbol>
        <symbol id="a-bell" viewBox="0 0 24 24">
          <path d="M5 17h14l-2-3V9A5 5 0 0 0 7 9v5l-2 3Zm5 3h4M12 2v2" />
        </symbol>
        <symbol id="a-clock" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 7v5l3 2" />
        </symbol>
        <symbol id="a-check" viewBox="0 0 24 24">
          <path d="m5 12 4 4L19 6" />
        </symbol>
        <symbol id="a-sort" viewBox="0 0 24 24">
          <path d="m8 9 4-4 4 4m-8 6 4 4 4-4" />
        </symbol>
        <symbol id="a-chevron" viewBox="0 0 24 24">
          <path d="m6 9 6 6 6-6" />
        </symbol>
        <symbol id="a-logout" viewBox="0 0 24 24">
          <path d="M10 4H4v16h6m-1-8h12m-4-4 4 4-4 4" />
        </symbol>
      </svg>
      <div className="admin-shell">
        <aside className="admin-sidebar" aria-label="Staff navigation">
          <a className="admin-brand" href="/admin">
            <span className="brand-mark" aria-hidden="true">
              <svg className="a-icon">
                <use href="#a-shield" />
              </svg>
            </span>
            <span>
              Agrilinks Nigeria<small>THE ADMIN WORKSPACE</small>
            </span>
          </a>
          <button
            type="button"
            className="admin-menu-toggle"
            aria-controls="admin-navigation"
            aria-expanded="false"
            aria-label="Open navigation"
          >
            <span></span>
            <span></span>
          </button>
          <nav id="admin-navigation" aria-label="Admin sections">
            <p className="nav-caption">PLATFORM MANAGEMENT</p>
            {truth(data?.Navigation)
              ? items(data?.Navigation).map((item1, index1) => (
                  <Fragment key={index1}>
                    {truth(item1?.Key === "audit") ? (
                      <>
                        <p className="nav-caption nav-caption-secondary">
                          GOVERNANCE &amp; ACCESS
                        </p>
                      </>
                    ) : null}
                    <a
                      href={text(item1?.Path)}
                      title={text(item1?.Title)}
                      className={
                        data?.Section?.Key === item1?.Key
                          ? "selected"
                          : undefined
                      }
                      aria-current={
                        data?.Section?.Key === item1?.Key ? "page" : undefined
                      }
                    >
                      <svg className="a-icon" aria-hidden="true">
                        <use href={"#a-" + text(item1?.Key)} />
                      </svg>
                      <span>
                        {truth(item1?.Key === "overview") ? (
                          <>Overview</>
                        ) : truth(item1?.Key === "users") ? (
                          <>Members &amp; farmers</>
                        ) : truth(item1?.Key === "marketplace") ? (
                          <>Marketplace</>
                        ) : truth(item1?.Key === "orders") ? (
                          <>Orders &amp; deliveries</>
                        ) : truth(item1?.Key === "finance") ? (
                          <>Finance &amp; wallets</>
                        ) : truth(item1?.Key === "support") ? (
                          <>Customer care</>
                        ) : truth(item1?.Key === "reports") ? (
                          <>Analytics</>
                        ) : truth(item1?.Key === "audit") ? (
                          <>Audit trail</>
                        ) : truth(item1?.Key === "staff") ? (
                          <>Team &amp; access</>
                        ) : truth(item1?.Key === "settings") ? (
                          <>Settings</>
                        ) : null}
                      </span>
                      {truth(data?.Section?.Key === item1?.Key) ? (
                        <>
                          <span className="nav-dot" aria-hidden="true"></span>
                        </>
                      ) : null}
                    </a>
                  </Fragment>
                ))
              : null}
            <a className="mobile-nav-footer" href="/dashboard">
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-up-right" />
              </svg>
              Member dashboard
            </a>
            <a className="mobile-nav-footer" href="/logout">
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-logout" />
              </svg>
              Log out
            </a>
            {truth(data?.User?.CanAdmin?.["revenue"]) ? (
              <>
                <a href="/admin/revenue">Platform revenue &amp; settlements</a>
              </>
            ) : null}
          </nav>
          <div className="sidebar-footer">
            <div className="sidebar-note">
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-leaf" />
              </svg>
              <p>
                Good stewardship.
                <br />
                <em>Great possibilities.</em>
              </p>
              <small>
                A clearer view of the people
                <br />
                and progress that matter.
              </small>
            </div>
            <a className="sidebar-member" href="/profile">
              <span className="admin-avatar" data-admin-initials="">
                AS
              </span>
              <span>
                <strong>{text(data?.User?.FullName)}</strong>
                <small>{text(humanStatus(data?.User?.AdminRole))}</small>
              </span>
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-up-right" />
              </svg>
            </a>
            <a className="sidebar-dashboard" href="/dashboard">
              Member dashboard
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-up-right" />
              </svg>
            </a>
            <a className="sidebar-logout" href="/logout">
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-logout" />
              </svg>
              Log out
            </a>
          </div>
        </aside>
        <div className="admin-main">
          <header className="admin-topbar">
            <span className="admin-breadcrumb">
              Workspace <span aria-hidden="true">/</span>
              <strong>
                {truth(data?.Section?.Key === "overview") ? (
                  <>Overview</>
                ) : truth(data?.Section?.Key === "users") ? (
                  <>Members &amp; farmers</>
                ) : truth(data?.Section?.Key === "marketplace") ? (
                  <>Marketplace</>
                ) : truth(data?.Section?.Key === "orders") ? (
                  <>Orders &amp; deliveries</>
                ) : truth(data?.Section?.Key === "finance") ? (
                  <>Finance &amp; wallets</>
                ) : truth(data?.Section?.Key === "support") ? (
                  <>Customer care</>
                ) : truth(data?.Section?.Key === "reports") ? (
                  <>Analytics</>
                ) : truth(data?.Section?.Key === "audit") ? (
                  <>Audit trail</>
                ) : truth(data?.Section?.Key === "staff") ? (
                  <>Team &amp; access</>
                ) : truth(data?.Section?.Key === "settings") ? (
                  <>Settings</>
                ) : null}
              </strong>
            </span>
            <button
              type="button"
              className="workspace-finder-trigger admin-js-control"
              aria-haspopup="dialog"
            >
              <svg className="a-icon" aria-hidden="true">
                <use href="#a-search" />
              </svg>
              <span>Find a workspace</span>
              <kbd data-shortcut-label="">Ctrl K</kbd>
            </button>
            <div className="admin-topbar-actions">
              <time id="admin-date"></time>
              <span className="staff-pill">
                <svg className="a-icon" aria-hidden="true">
                  <use href="#a-shield" />
                </svg>
                {text(humanStatus(data?.User?.AdminRole))}
              </span>
              <a
                className="admin-avatar"
                href="/profile"
                aria-label="Your profile"
                data-admin-initials=""
              >
                AS
              </a>
            </div>
          </header>
          <main id="admin-content">
            <div
              className={
                "page-heading " +
                (truth(data?.Section?.Key === "overview")
                  ? "overview-heading"
                  : "")
              }
            >
              <div>
                <p className="eyebrow">
                  <span></span>
                  {truth(data?.Section?.Key === "overview") ? (
                    <>YOUR PLATFORM. A WORLD OF POSSIBILITY.</>
                  ) : (
                    <>YOUR OPERATIONS, IN FOCUS</>
                  )}
                </p>
                <h1>
                  {truth(data?.Section?.Key === "overview") ? (
                    <>
                      A clearer view.
                      <br className="heading-mobile-break" />
                      <em>A stronger platform.</em>
                    </>
                  ) : (
                    <>{text(data?.Section?.Title)}</>
                  )}
                </h1>
                <p>
                  {truth(data?.Section?.Key === "overview") ? (
                    <>
                      <span id="admin-greeting">Welcome back</span>,{" "}
                      <strong id="admin-owner-name">
                        {text(data?.User?.FullName)}
                      </strong>
                      . Here's what is happening across Agrilinks Nigeria.
                    </>
                  ) : (
                    <>{text(data?.Section?.Description)}</>
                  )}
                </p>
              </div>
              <a className="button secondary" href={text(data?.Section?.Path)}>
                <svg className="a-icon" aria-hidden="true">
                  <use href="#a-refresh" />
                </svg>
                Refresh records
              </a>
            </div>
            {truth(data?.Notice) ? (
              <>
                <p className="notice" role="status">
                  <svg className="a-icon" aria-hidden="true">
                    <use href="#a-check" />
                  </svg>
                  {text(data?.Notice)}
                </p>
              </>
            ) : null}
            {truth(data?.Error) ? (
              <>
                <p className="error" role="alert">
                  <svg className="a-icon" aria-hidden="true">
                    <use href="#a-bell" />
                  </svg>
                  {text(data?.Error)}
                </p>
              </>
            ) : null}
            {truth(data?.Policies) ? (
              <>
                <section
                  className="panel policy-panel"
                  aria-label="Operational policies"
                >
                  <details>
                    <summary>
                      <svg className="a-icon" aria-hidden="true">
                        <use href="#a-shield" />
                      </svg>
                      Workspace policies<span>View guidance</span>
                      <svg className="a-icon" aria-hidden="true">
                        <use href="#a-chevron" />
                      </svg>
                    </summary>
                    <div>
                      {truth(data?.Policies)
                        ? items(data?.Policies).map((item2, index2) => (
                            <Fragment key={index2}>
                              <p>
                                <strong>{text(item2?.Label)}:</strong>
                                {text(item2?.Value)}
                              </p>
                            </Fragment>
                          ))
                        : null}
                    </div>
                  </details>
                </section>
              </>
            ) : null}
            {truth(data?.Metrics) ? (
              <>
                <section className="metrics" aria-label="Platform statistics">
                  {truth(data?.Metrics)
                    ? items(data?.Metrics).map((item3, index3) => (
                        <Fragment key={index3}>
                          {truth(
                            truth(item3?.Label === "Total users") ||
                              truth(item3?.Label === "Active farmers") ||
                              truth(item3?.Label === "Orders") ||
                              truth(item3?.Label === "Revenue") ||
                              truth(
                                truth(item3?.Label === "Buyers") &&
                                  truth(!truth(data?.User?.CanManageUsers)),
                              ),
                          ) ? (
                            <>
                              <article
                                className="metric"
                                data-metric={text(item3?.Label)}
                              >
                                <div className="metric-top">
                                  <p>{text(item3?.Label)}</p>
                                  <span className="metric-icon">
                                    <svg className="a-icon" aria-hidden="true">
                                      <use
                                        href={
                                          truth(
                                            item3?.Label === "Active farmers",
                                          )
                                            ? "#a-leaf"
                                            : truth(item3?.Label === "Orders")
                                              ? "#a-orders"
                                              : truth(
                                                    item3?.Label === "Revenue",
                                                  )
                                                ? "#a-finance"
                                                : "#a-users"
                                        }
                                      />
                                    </svg>
                                  </span>
                                </div>
                                <strong data-metric-value="">
                                  {text(item3?.Value)}
                                </strong>
                                <small>{text(item3?.Detail)}</small>
                                {truth(item3?.Path) ? (
                                  <>
                                    <a
                                      href={text(item3?.Path)}
                                      aria-label={"View " + text(item3?.Label)}
                                    >
                                      View records
                                      <svg
                                        className="a-icon"
                                        aria-hidden="true"
                                      >
                                        <use href="#a-up-right" />
                                      </svg>
                                    </a>
                                  </>
                                ) : null}
                              </article>
                            </>
                          ) : null}
                        </Fragment>
                      ))
                    : null}
                </section>
                <div className="overview-focus-grid">
                  <section
                    className="attention-panel"
                    aria-labelledby="attention-title"
                  >
                    <div className="attention-heading">
                      <div>
                        <p className="eyebrow">
                          A LITTLE ATTENTION GOES A LONG WAY
                        </p>
                        <h2 id="attention-title">Keep good things moving.</h2>
                      </div>
                      <span className="attention-emblem">
                        <svg className="a-icon" aria-hidden="true">
                          <use href="#a-bell" />
                        </svg>
                      </span>
                    </div>
                    <div className="attention-items">
                      {truth(data?.Metrics)
                        ? items(data?.Metrics).map((item4, index4) => (
                            <Fragment key={index4}>
                              {truth(
                                truth(item4?.Label === "Pending tasks") ||
                                  truth(item4?.Label === "Alerts") ||
                                  truth(item4?.Label === "Overdue support") ||
                                  truth(item4?.Label === "Payment requests"),
                              ) ? (
                                <>
                                  <a
                                    className="attention-item"
                                    href={text(item4?.Path)}
                                    data-attention={text(item4?.Label)}
                                  >
                                    <span className="attention-icon">
                                      <svg
                                        className="a-icon"
                                        aria-hidden="true"
                                      >
                                        <use
                                          href={
                                            truth(
                                              item4?.Label === "Pending tasks",
                                            )
                                              ? "#a-audit"
                                              : truth(
                                                    item4?.Label ===
                                                      "Overdue support",
                                                  )
                                                ? "#a-clock"
                                                : truth(
                                                      item4?.Label ===
                                                        "Payment requests",
                                                    )
                                                  ? "#a-finance"
                                                  : "#a-bell"
                                          }
                                        />
                                      </svg>
                                    </span>
                                    <span className="attention-copy">
                                      <strong>{text(item4?.Label)}</strong>
                                      <small>{text(item4?.Detail)}</small>
                                    </span>
                                    <span
                                      className="attention-count"
                                      data-metric-value=""
                                    >
                                      {text(item4?.Value)}
                                    </span>
                                    <svg
                                      className="a-icon attention-arrow"
                                      aria-hidden="true"
                                    >
                                      <use href="#a-arrow" />
                                    </svg>
                                  </a>
                                </>
                              ) : null}
                            </Fragment>
                          ))
                        : null}
                    </div>
                    <p className="attention-footnote">
                      Counts reflect saved platform records.
                    </p>
                  </section>
                  <section
                    className="platform-pulse"
                    aria-labelledby="pulse-title"
                  >
                    <svg
                      className="pulse-art"
                      viewBox="0 0 180 180"
                      aria-hidden="true"
                    >
                      <circle cx="100" cy="80" r="62" />
                      <circle cx="100" cy="80" r="49" />
                      <path d="M95 155V49m0 70c-43 0-51-26-52-44 36 0 52 24 52 44Zm0-29c42 0 53-22 55-46-35 2-55 22-55 46Zm0-15c-26 0-32-18-32-39 21 3 32 17 32 39M72 157h46" />
                    </svg>
                    <p className="eyebrow">ROOTED IN POSSIBILITY</p>
                    <h2 id="pulse-title">
                      People. Produce.
                      <br />
                      <em>Progress.</em>
                    </h2>
                    <p className="pulse-copy">
                      The community behind every good harvest.
                    </p>
                    <div className="pulse-metrics">
                      {truth(data?.Metrics)
                        ? items(data?.Metrics).map((item5, index5) => (
                            <Fragment key={index5}>
                              {truth(
                                truth(item5?.Label === "Buyers") ||
                                  truth(item5?.Label === "Listings"),
                              ) ? (
                                <>
                                  <div>
                                    <strong data-metric-value="">
                                      {text(item5?.Value)}
                                    </strong>
                                    <span>{text(item5?.Label)}</span>
                                    {truth(item5?.Path) ? (
                                      <>
                                        <a
                                          href={text(item5?.Path)}
                                          aria-label={
                                            "View " + text(item5?.Label)
                                          }
                                        >
                                          <svg
                                            className="a-icon"
                                            aria-hidden="true"
                                          >
                                            <use href="#a-up-right" />
                                          </svg>
                                        </a>
                                      </>
                                    ) : null}
                                  </div>
                                </>
                              ) : null}
                            </Fragment>
                          ))
                        : null}
                    </div>
                    <a className="pulse-link" href="/admin/users">
                      Meet your growing community
                      <svg className="a-icon" aria-hidden="true">
                        <use href="#a-arrow" />
                      </svg>
                    </a>
                  </section>
                </div>
              </>
            ) : null}
            {truth(data?.Section?.Key === "overview") ? (
              <>
                <section
                  className="workspace-shortcuts"
                  aria-label="Workspace shortcuts"
                >
                  <div className="section-caption">
                    <h2>Your next move, one click away.</h2>
                    <span>
                      {text(len(data?.Navigation))} workspaces at your
                      fingertips
                    </span>
                  </div>
                  <div className="shortcut-grid">
                    {truth(data?.Navigation)
                      ? items(data?.Navigation).map((item6, index6) => (
                          <Fragment key={index6}>
                            {truth(
                              truth(item6?.Key === "users") ||
                                truth(item6?.Key === "support") ||
                                truth(item6?.Key === "finance"),
                            ) ? (
                              <>
                                <a
                                  className="shortcut-card"
                                  href={text(item6?.Path)}
                                >
                                  <span className="shortcut-icon">
                                    <svg className="a-icon" aria-hidden="true">
                                      <use href={"#a-" + text(item6?.Key)} />
                                    </svg>
                                  </span>
                                  <span>
                                    <strong>
                                      {truth(item6?.Key === "overview") ? (
                                        <>Overview</>
                                      ) : truth(item6?.Key === "users") ? (
                                        <>Members &amp; farmers</>
                                      ) : truth(
                                          item6?.Key === "marketplace",
                                        ) ? (
                                        <>Marketplace</>
                                      ) : truth(item6?.Key === "orders") ? (
                                        <>Orders &amp; deliveries</>
                                      ) : truth(item6?.Key === "finance") ? (
                                        <>Finance &amp; wallets</>
                                      ) : truth(item6?.Key === "support") ? (
                                        <>Customer care</>
                                      ) : truth(item6?.Key === "reports") ? (
                                        <>Analytics</>
                                      ) : truth(item6?.Key === "audit") ? (
                                        <>Audit trail</>
                                      ) : truth(item6?.Key === "staff") ? (
                                        <>Team &amp; access</>
                                      ) : truth(item6?.Key === "settings") ? (
                                        <>Settings</>
                                      ) : null}
                                    </strong>
                                    <small>
                                      {truth(item6?.Key === "users") ? (
                                        <>People at the heart of the platform</>
                                      ) : truth(item6?.Key === "support") ? (
                                        <>A helping hand, right on time</>
                                      ) : (
                                        <>Every money movement, in view</>
                                      )}
                                    </small>
                                  </span>
                                  <svg className="a-icon" aria-hidden="true">
                                    <use href="#a-arrow" />
                                  </svg>
                                </a>
                              </>
                            ) : null}
                          </Fragment>
                        ))
                      : null}
                  </div>
                </section>
              </>
            ) : null}
            {truth(data?.Section?.Key === "staff") ? (
              <>
                <section className="panel">
                  <div className="panel-heading">
                    <h2>Roles &amp; access</h2>
                    <span className="subtle">
                      Permissions are enforced on every request
                    </span>
                  </div>
                  <div className="role-grid">
                    <article>
                      <h3>Super Admin</h3>
                      <p>
                        All ten sections. Manages staff permissions, platform
                        settings and audit records.
                      </p>
                    </article>
                    <article>
                      <h3>Customer Care</h3>
                      <p>
                        Overview, member records, orders and deliveries, support
                        tickets and disputes.
                      </p>
                    </article>
                    <article>
                      <h3>Operations Manager</h3>
                      <p>
                        Overview, users and verification, marketplace, orders,
                        finance, support and analytics.
                      </p>
                    </article>
                  </div>
                </section>
                <div className="form-panels">
                  <section className="panel">
                    <h2>Create a staff account</h2>
                    <p className="subtle">
                      Staff use the normal phone login. Assign only the access
                      their work needs.
                    </p>
                    <form
                      method="post"
                      action="/admin/action"
                      className="admin-form"
                    >
                      <input
                        type="hidden"
                        name="csrf"
                        defaultValue={text(data?.CSRF)}
                      />
                      <input
                        type="hidden"
                        name="return_to"
                        defaultValue={text(data?.Section?.Path)}
                      />
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="staff_create"
                      />
                      <label>
                        Full name
                        <input
                          name="name"
                          required={true}
                          maxLength="200"
                          autoComplete="name"
                        />
                      </label>
                      <label>
                        Phone number
                        <input
                          name="phone"
                          type="tel"
                          required={true}
                          placeholder="08012345678"
                          autoComplete="tel"
                        />
                      </label>
                      <label>
                        Email (optional)
                        <input name="email" type="email" autoComplete="email" />
                      </label>
                      <label>
                        Password
                        <input
                          name="password"
                          type="password"
                          required={true}
                          minLength="12"
                          maxLength="72"
                          autoComplete="new-password"
                        />
                        <small>
                          Use 12–72 bytes. Share credentials through your
                          approved staff onboarding process.
                        </small>
                      </label>
                      <label>
                        Staff role
                        <Select name="role" required={true}>
                          <option value="customer_care">Customer Care</option>
                          <option value="operations_manager">
                            Operations Manager
                          </option>
                          <option value="finance_admin">
                            Finance Administrator
                          </option>
                          <option value="senior_finance_admin">
                            Senior Finance Administrator
                          </option>
                          <option value="super_admin">Super Admin</option>
                        </Select>
                      </label>
                      <button>Create staff account</button>
                    </form>
                  </section>
                  <section className="panel">
                    <h2>Give an existing account staff access</h2>
                    <p className="subtle">
                      Find the member ID under Users &amp; Farmer Management.
                      Their member role is preserved.
                    </p>
                    <form
                      method="post"
                      action="/admin/action"
                      className="admin-form"
                    >
                      <input
                        type="hidden"
                        name="csrf"
                        defaultValue={text(data?.CSRF)}
                      />
                      <input
                        type="hidden"
                        name="return_to"
                        defaultValue={text(data?.Section?.Path)}
                      />
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="staff_role"
                      />
                      <label>
                        Member ID
                        <input
                          name="id"
                          type="number"
                          min="1"
                          required={true}
                        />
                      </label>
                      <label>
                        Staff role
                        <Select name="role">
                          <option value="customer_care">Customer Care</option>
                          <option value="operations_manager">
                            Operations Manager
                          </option>
                          <option value="finance_admin">
                            Finance Administrator
                          </option>
                          <option value="senior_finance_admin">
                            Senior Finance Administrator
                          </option>
                          <option value="super_admin">Super Admin</option>
                        </Select>
                      </label>
                      <label>
                        Reason
                        <Textarea
                          name="reason"
                          required={true}
                          maxLength="2000"
                        ></Textarea>
                      </label>
                      <button>Assign staff role</button>
                    </form>
                    <p className="subtle">
                      Disabling staff revokes their sessions and blocks login.
                      Another Super Admin must change your own access.
                    </p>
                  </section>
                </div>
              </>
            ) : null}
            {truth(data?.Section?.Key === "support") ? (
              <>
                <section className="panel">
                  <details className="create-details">
                    <summary>
                      Create a support ticket, complaint or dispute
                    </summary>
                    <form
                      method="post"
                      action="/admin/action"
                      className="admin-form form-grid"
                    >
                      <input
                        type="hidden"
                        name="csrf"
                        defaultValue={text(data?.CSRF)}
                      />
                      <input
                        type="hidden"
                        name="return_to"
                        defaultValue={text(data?.Section?.Path)}
                      />
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="ticket_create"
                      />
                      <label>
                        Member ID
                        <input
                          name="member_id"
                          type="number"
                          min="1"
                          required={true}
                        />
                      </label>
                      <label>
                        Related order ID (optional)
                        <input name="order_id" type="number" min="1" />
                      </label>
                      <label>
                        Type
                        <Select name="kind">
                          <option value="support">Support request</option>
                          <option value="complaint">Complaint</option>
                          <option value="dispute">Order dispute</option>
                          <option value="refund">Refund request</option>
                        </Select>
                      </label>
                      <label>
                        Priority
                        <Select name="priority">
                          <option value="">Platform default</option>
                          <option value="normal">Normal</option>
                          <option value="high">High</option>
                          <option value="urgent">Urgent</option>
                        </Select>
                      </label>
                      <label className="full-width">
                        Subject
                        <input name="subject" required={true} maxLength="200" />
                      </label>
                      <label className="full-width">
                        Description
                        <Textarea
                          name="description"
                          required={true}
                          maxLength="4000"
                        ></Textarea>
                      </label>
                      <button>Create ticket</button>
                    </form>
                  </details>
                  <p className="subtle">
                    Record refund requests here for investigation. Resolving a
                    ticket records the case outcome.
                  </p>
                </section>
              </>
            ) : null}
            {truth(
              truth(data?.Section?.Key === "users") ||
                truth(data?.Section?.Key === "marketplace"),
            ) ? (
              <>
                <section className="panel">
                  <details className="create-details">
                    <summary>
                      Report an account
                      {truth(data?.User?.CanManageUsers) ? (
                        <> or listing</>
                      ) : null}
                    </summary>
                    <form
                      method="post"
                      action="/admin/action"
                      className="admin-form form-grid"
                    >
                      <input
                        type="hidden"
                        name="csrf"
                        defaultValue={text(data?.CSRF)}
                      />
                      <input
                        type="hidden"
                        name="return_to"
                        defaultValue={text(data?.Section?.Path)}
                      />
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="report_create"
                      />
                      <label>
                        Report type
                        <Select name="target_kind">
                          <option value="user">Account</option>
                          {truth(data?.User?.CanManageUsers) ? (
                            <>
                              <option value="product">Listing</option>
                            </>
                          ) : null}
                        </Select>
                      </label>
                      <label>
                        Account or listing ID
                        <input
                          name="target_id"
                          type="number"
                          min="1"
                          required={true}
                        />
                      </label>
                      <label className="full-width">
                        Reason
                        <Textarea
                          name="reason"
                          required={true}
                          maxLength="2000"
                        ></Textarea>
                      </label>
                      <button>Record report</button>
                    </form>
                  </details>
                </section>
              </>
            ) : null}
            {truth(data?.Section?.Key === "settings") ? (
              <>
                <section className="panel">
                  <h2>Approved operational settings</h2>
                  <p className="subtle">
                    Default priority and response targets apply to support
                    tickets. Policies appear in the relevant workspace.
                    Notification templates are used for ticket opened and
                    resolved messages.
                  </p>
                  <div className="settings-grid">
                    {truth(data?.Settings)
                      ? items(data?.Settings).map((item7, index7) => (
                          <Fragment key={index7}>
                            <form
                              method="post"
                              action="/admin/action"
                              className="admin-form setting-form"
                            >
                              <input
                                type="hidden"
                                name="csrf"
                                defaultValue={text(data?.CSRF)}
                              />
                              <input
                                type="hidden"
                                name="return_to"
                                defaultValue={text(data?.Section?.Path)}
                              />
                              <input
                                type="hidden"
                                name="action"
                                defaultValue="setting_update"
                              />
                              <input
                                type="hidden"
                                name="key"
                                defaultValue={text(item7?.Key)}
                              />
                              <label>
                                {text(item7?.Label)}
                                {truth(
                                  item7?.Key === "support_default_priority",
                                ) ? (
                                  <>
                                    <Select name="value">
                                      <option
                                        value="normal"
                                        selected={truth(
                                          item7?.Value === "normal",
                                        )}
                                      >
                                        Normal
                                      </option>
                                      <option
                                        value="high"
                                        selected={truth(
                                          item7?.Value === "high",
                                        )}
                                      >
                                        High
                                      </option>
                                      <option
                                        value="urgent"
                                        selected={truth(
                                          item7?.Value === "urgent",
                                        )}
                                      >
                                        Urgent
                                      </option>
                                    </Select>
                                  </>
                                ) : truth(
                                    item7?.Key === "support_response_hours",
                                  ) ? (
                                  <>
                                    <input
                                      name="value"
                                      type="number"
                                      min="1"
                                      max="720"
                                      required={true}
                                      defaultValue={text(item7?.Value)}
                                    />
                                  </>
                                ) : (
                                  <>
                                    <Textarea
                                      name="value"
                                      required={true}
                                      maxLength="4000"
                                    >
                                      {text(item7?.Value)}
                                    </Textarea>
                                  </>
                                )}
                              </label>
                              <label>
                                Reason for change
                                <input
                                  name="reason"
                                  required={true}
                                  maxLength="2000"
                                />
                              </label>
                              <button>Save setting</button>
                            </form>
                          </Fragment>
                        ))
                      : null}
                  </div>
                </section>
              </>
            ) : null}
            {truth(data?.Section?.Key === "finance") ? (
              <>
                <section className="panel">
                  <details className="create-details">
                    <summary>Check a missing bank transfer</summary>
                    <p className="subtle">
                      Use the Flutterwave transaction ID to recover a transfer
                      missing from a member's funding history. The provider must
                      confirm the account, amount and payment status. Rechecking
                      a credited transfer does not credit it again.
                    </p>
                    <form
                      method="post"
                      action="/admin/action"
                      className="admin-form form-grid"
                    >
                      <input
                        type="hidden"
                        name="csrf"
                        defaultValue={text(data?.CSRF)}
                      />
                      <input
                        type="hidden"
                        name="return_to"
                        defaultValue={text(data?.Section?.Path)}
                      />
                      <input
                        type="hidden"
                        name="action"
                        defaultValue="wallet_funding_check"
                      />
                      <label>
                        Flutterwave transaction ID
                        <input
                          name="id"
                          inputMode="numeric"
                          pattern="[0-9]+"
                          maxLength="19"
                          required={true}
                        />
                      </label>
                      <label>
                        Investigation notes
                        <input name="reason" required={true} maxLength="2000" />
                      </label>
                      <button>Verify bank transfer</button>
                    </form>
                  </details>
                </section>
              </>
            ) : null}
            {truth(data?.Tables) ? (
              <>
                <div className="records-tools">
                  <form
                    className="search-form"
                    method="get"
                    action={text(data?.Section?.Path)}
                  >
                    <label htmlFor="admin-search">
                      Search all{" "}
                      {truth(data?.Section?.Key === "overview") ? (
                        <>overview</>
                      ) : (
                        <>
                          {truth(data?.Section?.Key === "overview") ? (
                            <>Overview</>
                          ) : truth(data?.Section?.Key === "users") ? (
                            <>Members &amp; farmers</>
                          ) : truth(data?.Section?.Key === "marketplace") ? (
                            <>Marketplace</>
                          ) : truth(data?.Section?.Key === "orders") ? (
                            <>Orders &amp; deliveries</>
                          ) : truth(data?.Section?.Key === "finance") ? (
                            <>Finance &amp; wallets</>
                          ) : truth(data?.Section?.Key === "support") ? (
                            <>Customer care</>
                          ) : truth(data?.Section?.Key === "reports") ? (
                            <>Analytics</>
                          ) : truth(data?.Section?.Key === "audit") ? (
                            <>Audit trail</>
                          ) : truth(data?.Section?.Key === "staff") ? (
                            <>Team &amp; access</>
                          ) : truth(data?.Section?.Key === "settings") ? (
                            <>Settings</>
                          ) : null}
                        </>
                      )}{" "}
                      records
                    </label>
                    <div>
                      <span className="record-search-input">
                        <svg className="a-icon" aria-hidden="true">
                          <use href="#a-search" />
                        </svg>
                        <input
                          id="admin-search"
                          type="search"
                          name="q"
                          defaultValue={text(data?.Search)}
                          maxLength="200"
                          placeholder="Search by ID, name or status"
                        />
                      </span>
                      <button type="submit">Search</button>
                      {truth(data?.Search) ? (
                        <>
                          <a
                            className="clear-search"
                            href={text(data?.Section?.Path)}
                          >
                            Clear
                          </a>
                        </>
                      ) : null}
                    </div>
                  </form>
                  <label className="table-jump admin-js-control">
                    Jump to records
                    <Select id="admin-table-jump">
                      <option value="">Choose a table</option>
                      {truth(data?.Tables)
                        ? items(data?.Tables).map((item8, index8) => (
                            <Fragment key={index8}>
                              <option value={"admin-table-" + text(item8?.Key)}>
                                {text(item8?.Title)}
                              </option>
                            </Fragment>
                          ))
                        : null}
                    </Select>
                  </label>
                </div>
              </>
            ) : null}
            {truth(data?.Tables)
              ? items(data?.Tables).map((item9, index9) => (
                  <Fragment key={index9}>
                    <section
                      className="panel records-panel"
                      id={"admin-table-" + text(item9?.Key)}
                      data-table-key={text(item9?.Key)}
                    >
                      <div className="panel-heading">
                        <div>
                          <p className="eyebrow">THE DETAILS THAT MATTER</p>
                          <h2>{text(item9?.Title)}</h2>
                        </div>
                        <div className="record-table-controls">
                          <span className="record-count" aria-live="polite">
                            {text(len(item9?.Rows))} records on this page
                          </span>
                          {truth(item9?.Rows) ? (
                            <>
                              <label className="record-status-filter admin-js-control">
                                <span className="sr-only">
                                  Filter {text(item9?.Title)} on this page by
                                  status
                                </span>
                                <Select data-table-status="">
                                  <option value="">
                                    All statuses · this page
                                  </option>
                                </Select>
                              </label>
                            </>
                          ) : null}
                        </div>
                      </div>
                      {truth(item9?.Rows) ? (
                        <>
                          <div
                            className="table-scroll"
                            tabIndex="0"
                            role="region"
                            aria-label={text(item9?.Title)}
                          >
                            <table>
                              <thead>
                                <tr>
                                  {truth(item9?.Columns)
                                    ? items(item9?.Columns).map(
                                        (item10, index10) => (
                                          <Fragment key={index10}>
                                            <th
                                              scope="col"
                                              data-column={text(index10)}
                                            >
                                              <span className="column-label">
                                                {text(item10)}
                                              </span>
                                            </th>
                                          </Fragment>
                                        ),
                                      )
                                    : null}
                                  {truth(
                                    truth(item9?.Key === "users") ||
                                      truth(item9?.Key === "products") ||
                                      truth(item9?.Key === "moderation") ||
                                      truth(item9?.Key === "tickets") ||
                                      truth(item9?.Key === "staff") ||
                                      truth(item9?.Key === "withdrawals") ||
                                      truth(item9?.Key === "transfers"),
                                  ) ? (
                                    <>
                                      <th scope="col">Actions</th>
                                    </>
                                  ) : null}
                                </tr>
                              </thead>
                              <tbody>
                                {truth(item9?.Rows)
                                  ? items(item9?.Rows).map(
                                      (item11, index11) => (
                                        <Fragment key={index11}>
                                          <tr
                                            data-status={text(item11?.Status)}
                                          >
                                            {truth(item11?.Cells)
                                              ? items(item11?.Cells).map(
                                                  (item12, index12) => (
                                                    <Fragment key={index12}>
                                                      <td>{text(item12)}</td>
                                                    </Fragment>
                                                  ),
                                                )
                                              : null}
                                            {truth(item9?.Key === "users") ? (
                                              <>
                                                <td>
                                                  {truth(
                                                    data?.User?.CanManageUsers,
                                                  ) ? (
                                                    <>
                                                      <details className="row-action">
                                                        <summary>
                                                          Manage account #
                                                          {text(item11?.ID)}
                                                        </summary>
                                                        <form
                                                          method="post"
                                                          action="/admin/action"
                                                          className="admin-form"
                                                        >
                                                          <input
                                                            type="hidden"
                                                            name="csrf"
                                                            defaultValue={text(
                                                              data?.CSRF,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="return_to"
                                                            defaultValue={text(
                                                              data?.Section
                                                                ?.Path,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="action"
                                                            defaultValue="user_status"
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="id"
                                                            defaultValue={text(
                                                              item11?.ID,
                                                            )}
                                                          />
                                                          <label>
                                                            Account status
                                                            <Select name="value">
                                                              <option
                                                                value="active"
                                                                selected={truth(
                                                                  item11?.Status ===
                                                                    "active",
                                                                )}
                                                              >
                                                                Active
                                                              </option>
                                                              <option
                                                                value="suspended"
                                                                selected={truth(
                                                                  item11?.Status ===
                                                                    "suspended",
                                                                )}
                                                              >
                                                                Suspended
                                                              </option>
                                                            </Select>
                                                          </label>
                                                          <label>
                                                            Reason
                                                            <Textarea
                                                              name="reason"
                                                              required={true}
                                                              maxLength="2000"
                                                            ></Textarea>
                                                          </label>
                                                          <button>
                                                            Save account status
                                                          </button>
                                                        </form>
                                                        <form
                                                          method="post"
                                                          action="/admin/action"
                                                          className="admin-form"
                                                        >
                                                          <input
                                                            type="hidden"
                                                            name="csrf"
                                                            defaultValue={text(
                                                              data?.CSRF,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="return_to"
                                                            defaultValue={text(
                                                              data?.Section
                                                                ?.Path,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="action"
                                                            defaultValue="user_verification"
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="id"
                                                            defaultValue={text(
                                                              item11?.ID,
                                                            )}
                                                          />
                                                          <label>
                                                            Verification status
                                                            <Select
                                                              name="value"
                                                              required={true}
                                                            >
                                                              <option value="">
                                                                Choose status
                                                              </option>
                                                              <option value="pending">
                                                                Pending review
                                                              </option>
                                                              <option value="verified">
                                                                Verified
                                                              </option>
                                                              <option value="rejected">
                                                                Rejected
                                                              </option>
                                                              <option value="unverified">
                                                                Unverified
                                                              </option>
                                                            </Select>
                                                          </label>
                                                          <label>
                                                            Review notes
                                                            <Textarea
                                                              name="reason"
                                                              required={true}
                                                              maxLength="2000"
                                                            ></Textarea>
                                                          </label>
                                                          <button>
                                                            Record verification
                                                          </button>
                                                        </form>
                                                      </details>
                                                    </>
                                                  ) : (
                                                    <>
                                                      <span className="subtle">
                                                        View only
                                                      </span>
                                                    </>
                                                  )}
                                                </td>
                                              </>
                                            ) : truth(
                                                item9?.Key === "products",
                                              ) ? (
                                              <>
                                                <td>
                                                  <details className="row-action">
                                                    <summary>
                                                      Manage listing #
                                                      {text(item11?.ID)}
                                                    </summary>
                                                    <form
                                                      method="post"
                                                      action="/admin/action"
                                                      className="admin-form"
                                                    >
                                                      <input
                                                        type="hidden"
                                                        name="csrf"
                                                        defaultValue={text(
                                                          data?.CSRF,
                                                        )}
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="return_to"
                                                        defaultValue={text(
                                                          data?.Section?.Path,
                                                        )}
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="action"
                                                        defaultValue="product_status"
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="id"
                                                        defaultValue={text(
                                                          item11?.ID,
                                                        )}
                                                      />
                                                      <label>
                                                        Listing status
                                                        <Select
                                                          name="value"
                                                          required={true}
                                                        >
                                                          <option value="">
                                                            Choose status
                                                          </option>
                                                          <option value="active">
                                                            Active
                                                          </option>
                                                          <option value="inactive">
                                                            Inactive
                                                          </option>
                                                        </Select>
                                                      </label>
                                                      <label>
                                                        Reason
                                                        <Textarea
                                                          name="reason"
                                                          required={true}
                                                          maxLength="2000"
                                                        ></Textarea>
                                                      </label>
                                                      <button>
                                                        Save listing status
                                                      </button>
                                                    </form>
                                                  </details>
                                                </td>
                                              </>
                                            ) : truth(
                                                item9?.Key === "moderation",
                                              ) ? (
                                              <>
                                                <td>
                                                  {truth(
                                                    data?.User?.CanManageUsers,
                                                  ) ? (
                                                    <>
                                                      <details className="row-action">
                                                        <summary>
                                                          Review report #
                                                          {text(item11?.ID)}
                                                        </summary>
                                                        <form
                                                          method="post"
                                                          action="/admin/action"
                                                          className="admin-form"
                                                        >
                                                          <input
                                                            type="hidden"
                                                            name="csrf"
                                                            defaultValue={text(
                                                              data?.CSRF,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="return_to"
                                                            defaultValue={text(
                                                              data?.Section
                                                                ?.Path,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="action"
                                                            defaultValue="report_update"
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="id"
                                                            defaultValue={text(
                                                              item11?.ID,
                                                            )}
                                                          />
                                                          <label>
                                                            Report status
                                                            <Select
                                                              name="value"
                                                              required={true}
                                                            >
                                                              <option value="">
                                                                Choose status
                                                              </option>
                                                              <option value="open">
                                                                Open
                                                              </option>
                                                              <option value="investigating">
                                                                Investigating
                                                              </option>
                                                              <option value="resolved">
                                                                Resolved
                                                              </option>
                                                              <option value="dismissed">
                                                                Dismissed
                                                              </option>
                                                            </Select>
                                                          </label>
                                                          <label>
                                                            Findings /
                                                            resolution
                                                            <Textarea
                                                              name="reason"
                                                              required={true}
                                                              maxLength="2000"
                                                            ></Textarea>
                                                          </label>
                                                          <button>
                                                            Record review
                                                          </button>
                                                        </form>
                                                      </details>
                                                    </>
                                                  ) : (
                                                    <>
                                                      <span className="subtle">
                                                        View only
                                                      </span>
                                                    </>
                                                  )}
                                                </td>
                                              </>
                                            ) : truth(
                                                item9?.Key === "tickets",
                                              ) ? (
                                              <>
                                                <td>
                                                  <a
                                                    className="text-link"
                                                    href={
                                                      "/admin/support?q=" +
                                                      text(item11?.ID)
                                                    }
                                                  >
                                                    Manage ticket #
                                                    {text(item11?.ID)}
                                                  </a>
                                                  {truth(
                                                    data?.Section?.Key ===
                                                      "support",
                                                  ) ? (
                                                    <>
                                                      <details className="row-action">
                                                        <summary>
                                                          Update ticket
                                                        </summary>
                                                        <form
                                                          method="post"
                                                          action="/admin/action"
                                                          className="admin-form"
                                                        >
                                                          <input
                                                            type="hidden"
                                                            name="csrf"
                                                            defaultValue={text(
                                                              data?.CSRF,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="return_to"
                                                            defaultValue={text(
                                                              data?.Section
                                                                ?.Path,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="action"
                                                            defaultValue="ticket_update"
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="id"
                                                            defaultValue={text(
                                                              item11?.ID,
                                                            )}
                                                          />
                                                          <label>
                                                            Status
                                                            <Select name="value">
                                                              <option
                                                                value="open"
                                                                selected={truth(
                                                                  item11?.Status ===
                                                                    "open",
                                                                )}
                                                              >
                                                                Open
                                                              </option>
                                                              <option
                                                                value="in_progress"
                                                                selected={truth(
                                                                  item11?.Status ===
                                                                    "in_progress",
                                                                )}
                                                              >
                                                                In progress
                                                              </option>
                                                              <option
                                                                value="resolved"
                                                                selected={truth(
                                                                  item11?.Status ===
                                                                    "resolved",
                                                                )}
                                                              >
                                                                Resolved
                                                              </option>
                                                              <option
                                                                value="closed"
                                                                selected={truth(
                                                                  item11?.Status ===
                                                                    "closed",
                                                                )}
                                                              >
                                                                Closed
                                                              </option>
                                                            </Select>
                                                          </label>
                                                          <label>
                                                            Assign staff
                                                            <Select name="assignee">
                                                              <option value="keep">
                                                                Keep current
                                                                assignment
                                                              </option>
                                                              <option value="">
                                                                Unassigned
                                                              </option>
                                                              {truth(
                                                                data?.Staff,
                                                              )
                                                                ? items(
                                                                    data?.Staff,
                                                                  ).map(
                                                                    (
                                                                      item13,
                                                                      index13,
                                                                    ) => (
                                                                      <Fragment
                                                                        key={
                                                                          index13
                                                                        }
                                                                      >
                                                                        <option
                                                                          value={text(
                                                                            item13?.ID,
                                                                          )}
                                                                        >
                                                                          {text(
                                                                            item13?.Name,
                                                                          )}{" "}
                                                                          ·{" "}
                                                                          {text(
                                                                            humanStatus(
                                                                              item13?.Role,
                                                                            ),
                                                                          )}
                                                                        </option>
                                                                      </Fragment>
                                                                    ),
                                                                  )
                                                                : null}
                                                            </Select>
                                                          </label>
                                                          <label>
                                                            Update / resolution
                                                            notes
                                                            <Textarea
                                                              name="reason"
                                                              required={true}
                                                              maxLength="2000"
                                                            ></Textarea>
                                                          </label>
                                                          <button>
                                                            Save ticket update
                                                          </button>
                                                        </form>
                                                      </details>
                                                    </>
                                                  ) : null}
                                                </td>
                                              </>
                                            ) : truth(
                                                item9?.Key === "staff",
                                              ) ? (
                                              <>
                                                <td>
                                                  <details className="row-action">
                                                    <summary>
                                                      Manage staff #
                                                      {text(item11?.ID)}
                                                    </summary>
                                                    <form
                                                      method="post"
                                                      action="/admin/action"
                                                      className="admin-form"
                                                    >
                                                      <input
                                                        type="hidden"
                                                        name="csrf"
                                                        defaultValue={text(
                                                          data?.CSRF,
                                                        )}
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="return_to"
                                                        defaultValue={text(
                                                          data?.Section?.Path,
                                                        )}
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="action"
                                                        defaultValue="staff_role"
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="id"
                                                        defaultValue={text(
                                                          item11?.ID,
                                                        )}
                                                      />
                                                      <label>
                                                        Staff role
                                                        <Select
                                                          name="role"
                                                          required={true}
                                                        >
                                                          <option value="">
                                                            Choose role
                                                          </option>
                                                          <option value="customer_care">
                                                            Customer Care
                                                          </option>
                                                          <option value="operations_manager">
                                                            Operations Manager
                                                          </option>
                                                          <option value="finance_admin">
                                                            Finance
                                                            Administrator
                                                          </option>
                                                          <option value="senior_finance_admin">
                                                            Senior Finance
                                                            Administrator
                                                          </option>
                                                          <option value="super_admin">
                                                            Super Admin
                                                          </option>
                                                        </Select>
                                                      </label>
                                                      <label>
                                                        Reason
                                                        <input
                                                          name="reason"
                                                          required={true}
                                                          maxLength="2000"
                                                        />
                                                      </label>
                                                      <button>
                                                        Change role
                                                      </button>
                                                    </form>
                                                    <form
                                                      method="post"
                                                      action="/admin/action"
                                                      className="admin-form"
                                                    >
                                                      <input
                                                        type="hidden"
                                                        name="csrf"
                                                        defaultValue={text(
                                                          data?.CSRF,
                                                        )}
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="return_to"
                                                        defaultValue={text(
                                                          data?.Section?.Path,
                                                        )}
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="action"
                                                        defaultValue="staff_access"
                                                      />
                                                      <input
                                                        type="hidden"
                                                        name="id"
                                                        defaultValue={text(
                                                          item11?.ID,
                                                        )}
                                                      />
                                                      <label>
                                                        Staff access
                                                        <Select name="value">
                                                          <option
                                                            value="enabled"
                                                            selected={truth(
                                                              item11?.Status ===
                                                                "enabled",
                                                            )}
                                                          >
                                                            Enabled
                                                          </option>
                                                          <option
                                                            value="disabled"
                                                            selected={truth(
                                                              item11?.Status ===
                                                                "disabled",
                                                            )}
                                                          >
                                                            Disabled
                                                          </option>
                                                        </Select>
                                                      </label>
                                                      <label>
                                                        Reason
                                                        <input
                                                          name="reason"
                                                          required={true}
                                                          maxLength="2000"
                                                        />
                                                      </label>
                                                      <button>
                                                        Save staff access
                                                      </button>
                                                    </form>
                                                  </details>
                                                </td>
                                              </>
                                            ) : truth(
                                                truth(
                                                  item9?.Key === "withdrawals",
                                                ) ||
                                                  truth(
                                                    item9?.Key === "transfers",
                                                  ),
                                              ) ? (
                                              <>
                                                <td>
                                                  {truth(
                                                    truth(
                                                      item11?.Status ===
                                                        "pending",
                                                    ) ||
                                                      truth(
                                                        truth(
                                                          item9?.Key ===
                                                            "withdrawals",
                                                        ) &&
                                                          truth(
                                                            item11?.Status ===
                                                              "processing",
                                                          ),
                                                      ),
                                                  ) ? (
                                                    <>
                                                      <details className="row-action">
                                                        <summary>
                                                          Review request #
                                                          {text(item11?.ID)}
                                                        </summary>
                                                        <form
                                                          method="post"
                                                          action="/admin/action"
                                                          className="admin-form"
                                                        >
                                                          <input
                                                            type="hidden"
                                                            name="csrf"
                                                            defaultValue={text(
                                                              data?.CSRF,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="return_to"
                                                            defaultValue={text(
                                                              data?.Section
                                                                ?.Path,
                                                            )}
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="action"
                                                            defaultValue={
                                                              truth(
                                                                item9?.Key ===
                                                                  "withdrawals",
                                                              )
                                                                ? "wallet_withdrawal"
                                                                : "wallet_transfer"
                                                            }
                                                          />
                                                          <input
                                                            type="hidden"
                                                            name="id"
                                                            defaultValue={text(
                                                              item11?.ID,
                                                            )}
                                                          />
                                                          <label>
                                                            Decision
                                                            <Select
                                                              name="value"
                                                              required={true}
                                                            >
                                                              <option value="">
                                                                Choose decision
                                                              </option>
                                                              {truth(
                                                                item9?.Key ===
                                                                  "withdrawals",
                                                              ) ? (
                                                                <>
                                                                  {truth(
                                                                    item11?.Status ===
                                                                      "pending",
                                                                  ) ? (
                                                                    <>
                                                                      <option value="approve">
                                                                        Check
                                                                        bank and
                                                                        approve
                                                                        payment
                                                                      </option>
                                                                      <option value="rejected">
                                                                        Reject
                                                                        and
                                                                        return
                                                                        money
                                                                      </option>
                                                                    </>
                                                                  ) : (
                                                                    <>
                                                                      <option value="check">
                                                                        Check
                                                                        payment
                                                                        status
                                                                      </option>
                                                                    </>
                                                                  )}
                                                                </>
                                                              ) : (
                                                                <>
                                                                  <option value="completed">
                                                                    Approve
                                                                    transfer
                                                                  </option>
                                                                  <option value="rejected">
                                                                    Reject and
                                                                    return money
                                                                  </option>
                                                                </>
                                                              )}
                                                            </Select>
                                                          </label>
                                                          <label>
                                                            Review notes
                                                            <Textarea
                                                              name="reason"
                                                              required={true}
                                                              maxLength="2000"
                                                            ></Textarea>
                                                          </label>
                                                          <button>
                                                            Submit payment
                                                            decision
                                                          </button>
                                                        </form>
                                                      </details>
                                                    </>
                                                  ) : (
                                                    <>
                                                      <span className="subtle">
                                                        {text(
                                                          humanStatus(
                                                            item11?.Status,
                                                          ),
                                                        )}
                                                      </span>
                                                    </>
                                                  )}
                                                </td>
                                              </>
                                            ) : null}
                                          </tr>
                                        </Fragment>
                                      ),
                                    )
                                  : null}
                              </tbody>
                            </table>
                          </div>
                          <div
                            className="empty-state filtered-empty admin-js-control"
                            hidden={true}
                          >
                            <span className="empty-emblem">
                              <svg className="a-icon" aria-hidden="true">
                                <use href="#a-search" />
                              </svg>
                            </span>
                            <h3>No records with this status on this page.</h3>
                            <p>
                              Choose another status or search all records above.
                            </p>
                            <button
                              type="button"
                              className="button secondary"
                              data-reset-table=""
                            >
                              Show this page's records
                            </button>
                          </div>
                          <p className="table-hint admin-js-control">
                            <svg className="a-icon" aria-hidden="true">
                              <use href="#a-sort" />
                            </svg>
                            Click a column to sort this page.
                            <span>Scroll sideways to see more details.</span>
                          </p>
                        </>
                      ) : (
                        <>
                          <div className="empty-state">
                            <span className="empty-emblem">
                              <svg className="a-icon" aria-hidden="true">
                                <use
                                  href={
                                    truth(data?.Search)
                                      ? "#a-search"
                                      : "#a-leaf"
                                  }
                                />
                              </svg>
                            </span>
                            <h3>
                              {truth(data?.Search) ? (
                                <>A different search might do the trick.</>
                              ) : (
                                <>A little quiet. A lot of possibility.</>
                              )}
                            </h3>
                            <p>
                              {truth(data?.Search) ? (
                                <>
                                  No records match your search. Try another
                                  name, ID or status.
                                </>
                              ) : (
                                <>
                                  New records will appear here as the platform
                                  grows.
                                </>
                              )}
                            </p>
                            {truth(data?.Search) ? (
                              <>
                                <a
                                  className="button secondary"
                                  href={text(data?.Section?.Path)}
                                >
                                  Clear search
                                </a>
                              </>
                            ) : null}
                          </div>
                        </>
                      )}
                    </section>
                  </Fragment>
                ))
              : null}
            {truth(data?.Tables) ? (
              <>
                <nav className="pagination" aria-label="Record pages">
                  {truth(data?.Page > 1) ? (
                    <>
                      <a
                        className="button secondary"
                        href={
                          text(data?.Section?.Path) +
                          "?q=" +
                          text(data?.Search) +
                          "&page=" +
                          text(data?.PreviousPage)
                        }
                      >
                        ← Previous
                      </a>
                    </>
                  ) : null}
                  <span>
                    Page {text(data?.Page)} · up to 25 records per table
                  </span>
                  {truth(data?.HasNext) ? (
                    <>
                      <a
                        className="button secondary"
                        href={
                          text(data?.Section?.Path) +
                          "?q=" +
                          text(data?.Search) +
                          "&page=" +
                          text(data?.NextPage)
                        }
                      >
                        Next →
                      </a>
                    </>
                  ) : null}
                </nav>
              </>
            ) : null}
            {truth(data?.Section?.Key === "overview") ? (
              <>
                <details className="workspace-directory">
                  <summary>
                    <svg className="a-icon" aria-hidden="true">
                      <use href="#a-overview" />
                    </svg>
                    Explore your workspaces
                    <span>
                      {text(len(data?.Navigation))} areas available to your role
                    </span>
                    <svg className="a-icon" aria-hidden="true">
                      <use href="#a-chevron" />
                    </svg>
                  </summary>
                  <section
                    className="section-directory"
                    aria-label="Management areas"
                  >
                    {truth(data?.Navigation)
                      ? items(data?.Navigation).map((item14, index14) => (
                          <Fragment key={index14}>
                            {truth(item14?.Key !== "overview") ? (
                              <>
                                <a
                                  className="directory-card"
                                  href={text(item14?.Path)}
                                >
                                  <span className="directory-icon">
                                    <svg className="a-icon" aria-hidden="true">
                                      <use href={"#a-" + text(item14?.Key)} />
                                    </svg>
                                  </span>
                                  <h2>
                                    {truth(item14?.Key === "overview") ? (
                                      <>Overview</>
                                    ) : truth(item14?.Key === "users") ? (
                                      <>Members &amp; farmers</>
                                    ) : truth(item14?.Key === "marketplace") ? (
                                      <>Marketplace</>
                                    ) : truth(item14?.Key === "orders") ? (
                                      <>Orders &amp; deliveries</>
                                    ) : truth(item14?.Key === "finance") ? (
                                      <>Finance &amp; wallets</>
                                    ) : truth(item14?.Key === "support") ? (
                                      <>Customer care</>
                                    ) : truth(item14?.Key === "reports") ? (
                                      <>Analytics</>
                                    ) : truth(item14?.Key === "audit") ? (
                                      <>Audit trail</>
                                    ) : truth(item14?.Key === "staff") ? (
                                      <>Team &amp; access</>
                                    ) : truth(item14?.Key === "settings") ? (
                                      <>Settings</>
                                    ) : null}
                                  </h2>
                                  <p>{text(item14?.Description)}</p>
                                  <svg
                                    className="a-icon directory-arrow"
                                    aria-hidden="true"
                                  >
                                    <use href="#a-up-right" />
                                  </svg>
                                </a>
                              </>
                            ) : null}
                          </Fragment>
                        ))
                      : null}
                  </section>
                </details>
              </>
            ) : null}
            <footer className="workspace-footer">
              <span>
                <svg className="a-icon" aria-hidden="true">
                  <use href="#a-shield" />
                </svg>
                Agrilinks Nigeria. Growing with good stewardship.
              </span>
              <span>Record times are shown in UTC</span>
            </footer>
          </main>
        </div>
      </div>
      <dialog
        id="admin-workspace-finder"
        className="workspace-finder"
        aria-labelledby="workspace-finder-title"
      >
        <div className="finder-heading">
          <div>
            <p className="eyebrow">RIGHT WHERE YOU NEED TO BE</p>
            <h2 id="workspace-finder-title">Find your workspace.</h2>
          </div>
          <button
            type="button"
            className="finder-close"
            aria-label="Close workspace finder"
          >
            ×
          </button>
        </div>
        <label className="finder-search">
          <svg className="a-icon" aria-hidden="true">
            <use href="#a-search" />
          </svg>
          <span className="sr-only">Search available workspaces</span>
          <input
            id="workspace-finder-search"
            type="search"
            placeholder="Members, finance, support…"
            autoComplete="off"
          />
        </label>
        <nav className="finder-results" aria-label="Available workspaces">
          {truth(data?.Navigation)
            ? items(data?.Navigation).map((item15, index15) => (
                <Fragment key={index15}>
                  <a href={text(item15?.Path)}>
                    <span className="finder-result-icon">
                      <svg className="a-icon" aria-hidden="true">
                        <use href={"#a-" + text(item15?.Key)} />
                      </svg>
                    </span>
                    <span>
                      <strong>
                        {truth(item15?.Key === "overview") ? (
                          <>Overview</>
                        ) : truth(item15?.Key === "users") ? (
                          <>Members &amp; farmers</>
                        ) : truth(item15?.Key === "marketplace") ? (
                          <>Marketplace</>
                        ) : truth(item15?.Key === "orders") ? (
                          <>Orders &amp; deliveries</>
                        ) : truth(item15?.Key === "finance") ? (
                          <>Finance &amp; wallets</>
                        ) : truth(item15?.Key === "support") ? (
                          <>Customer care</>
                        ) : truth(item15?.Key === "reports") ? (
                          <>Analytics</>
                        ) : truth(item15?.Key === "audit") ? (
                          <>Audit trail</>
                        ) : truth(item15?.Key === "staff") ? (
                          <>Team &amp; access</>
                        ) : truth(item15?.Key === "settings") ? (
                          <>Settings</>
                        ) : null}
                      </strong>
                      <small>{text(item15?.Description)}</small>
                    </span>
                    <svg className="a-icon" aria-hidden="true">
                      <use href="#a-arrow" />
                    </svg>
                  </a>
                </Fragment>
              ))
            : null}
        </nav>
        <p className="finder-empty" hidden={true}>
          No workspaces match. Try a different word.
        </p>
        <p className="finder-footnote">
          Only workspaces available to your role are shown.<kbd>Esc</kbd> to
          close
        </p>
      </dialog>
    </>
  );
}
