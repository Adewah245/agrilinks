import { WorkspaceSidebar } from "../components/Workspace.jsx";
import { Fragment } from "react";
import { truth, items, text } from "../lib/view.js";
export default function NotificationPage({ data }) {
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/notifications">
        <a href="/dashboard" className="nav-link">
          {" "}
          Dashboard{" "}
        </a>
        <a href="/storage" className="nav-link">
          {" "}
          Storage{" "}
        </a>
        <a href="/marketplace" className="nav-link">
          {" "}
          Marketplace{" "}
        </a>
        <a href="/negotiations" className="nav-link">
          {" "}
          Negotiations{" "}
        </a>
        <a href="/chat" className="nav-link">
          {" "}
          Chat{" "}
        </a>
        <a href="/cart" className="nav-link">
          {" "}
          Cart{" "}
        </a>
        <a href="/ai-assistant" className="nav-link">
          {" "}
          AI Assistant{" "}
        </a>
        <a href="/profile" className="nav-link">
          {" "}
          Profile{" "}
        </a>
        <a
          href="/notifications"
          className="nav-link notification-link"
          aria-current="page"
          aria-label="Notifications"
        >
          <span className="notification-bell" aria-hidden="true">
            {" "}
            🔔{" "}
          </span>
          <span className="notification-label"> Notifications </span>
          {truth(data?.UnreadCount > 0) ? (
            <>
              <span
                className="notification-badge"
                aria-label={text(data?.UnreadCount) + " unread notifications"}
              >
                {text(data?.UnreadCount)}
              </span>
            </>
          ) : null}
        </a>
        <a href="/logout" className="nav-link logout-link">
          {" "}
          Logout{" "}
        </a>
      </WorkspaceSidebar>
      <main className="notifications-page">
        <section
          className="notifications-container"
          aria-labelledby="notifications-title"
        >
          <div className="notifications-header">
            <div>
              <h1 id="notifications-title"> Notifications </h1>
              <p className="notifications-subtitle">
                {" "}
                Stay updated with important activities on Agrilinks
                Nigeria.{" "}
              </p>
            </div>
            {truth(data?.UnreadCount > 0) ? (
              <>
                <div className="notification-summary" aria-live="polite">
                  <span className="notification-summary-count">
                    {text(data?.UnreadCount)}
                  </span>
                  <span className="notification-summary-text">
                    {" "}
                    unread notification
                    {truth(data?.UnreadCount !== 1) ? <>s</> : null}
                  </span>
                </div>
              </>
            ) : null}
          </div>
          {truth(data?.Notifications) ? (
            <>
              <section
                className="notification-list"
                aria-label="Your notifications"
              >
                {truth(data?.Notifications)
                  ? items(data?.Notifications).map((item1, index1) => (
                      <Fragment key={index1}>
                        <article
                          className={
                            "notification-card " +
                            (truth(!truth(item1?.IsRead))
                              ? "notification-unread"
                              : "")
                          }
                          data-notification-id={text(item1?.ID)}
                        >
                          <div className="notification-icon-wrapper">
                            <span
                              className="notification-card-icon"
                              aria-hidden="true"
                            >
                              {" "}
                              🔔{" "}
                            </span>
                          </div>
                          <div className="notification-content">
                            <div className="notification-card-header">
                              <h2 className="notification-title">
                                {text(item1?.Title)}
                              </h2>
                              {truth(!truth(item1?.IsRead)) ? (
                                <>
                                  <span
                                    className="notification-unread-indicator"
                                    aria-label="Unread notification"
                                  >
                                    {" "}
                                    New{" "}
                                  </span>
                                </>
                              ) : null}
                            </div>
                            {truth(item1?.Type) ? (
                              <>
                                <span className="notification-type">
                                  {text(item1?.Type)}
                                </span>
                              </>
                            ) : null}
                            <p className="notification-message">
                              {text(item1?.Message)}
                            </p>
                            <time
                              className="notification-date"
                              dateTime={text(item1?.CreatedAt)}
                            >
                              {text(item1?.CreatedAt)}
                            </time>
                          </div>
                          <div className="notification-actions">
                            {truth(!truth(item1?.IsRead)) ? (
                              <>
                                <form
                                  method="POST"
                                  action="/notifications/read"
                                  className="notification-action-form"
                                >
                                  <input
                                    type="hidden"
                                    name="notification_id"
                                    defaultValue={text(item1?.ID)}
                                  />
                                  <button
                                    type="submit"
                                    className="notification-action-button"
                                    aria-label="Mark notification as read"
                                  >
                                    {" "}
                                    Mark as read{" "}
                                  </button>
                                </form>
                              </>
                            ) : null}
                          </div>
                        </article>
                      </Fragment>
                    ))
                  : null}
              </section>
            </>
          ) : (
            <>
              <section className="notifications-empty" aria-live="polite">
                <div className="notifications-empty-icon" aria-hidden="true">
                  {" "}
                  🔔{" "}
                </div>
                <h2> No notifications yet </h2>
                <p>
                  {" "}
                  When there is an important update about your products,
                  marketplace activity, negotiations, orders, messages, or other
                  Agrilinks Nigeria activities, it will appear here.{" "}
                </p>
                <a href="/dashboard" className="btn btn-primary">
                  {" "}
                  Go to Dashboard{" "}
                </a>
              </section>
            </>
          )}
        </section>
      </main>
      <footer className="site-footer">
        <p> © 2026 Agrilinks Nigeria. Supporting smarter agriculture. </p>
      </footer>
    </div>
  );
}
