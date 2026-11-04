import Navbar from "../components/Navbar.jsx";
import { Fragment } from "react";
import { truth, items, text, len } from "../lib/view.js";
import mountChat from "../behaviors/chat.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function ChatPage({ data }) {
  usePageBehaviors([mountChat], data);
  return (
    <>
      <Navbar className="navbar">
        <a
          className="logo"
          href="/dashboard"
          aria-label="Agrilinks Nigeria Dashboard"
        >
          <img
            src="/static/assets/logo/agrilinks-nigeria-logo.svg"
            alt="Agrilinks Nigeria"
          />
        </a>
        <nav aria-label="Main navigation">
          <a href="/dashboard"> Dashboard </a>
          <a href="/storage"> Storage </a>
          <a href="/marketplace"> Marketplace </a>
          <a href="/negotiations"> Negotiations </a>
          <a href="/ai-assistant"> AI Assistant </a>
          <a href="/chat" aria-current="page">
            {" "}
            Chat{" "}
            {truth(data?.UnreadCount > 0) ? (
              <>
                <span
                  className="chat-nav-unread"
                  aria-label={text(data?.UnreadCount) + " unread messages"}
                >
                  {text(data?.UnreadCount)}
                </span>
              </>
            ) : null}
          </a>
          <a href="/profile"> Profile </a>
          <a href="/notifications"> Notifications </a>
          <a href="/logout"> Logout </a>
        </nav>
      </Navbar>
      <main className="chat-container">
        <header className="chat-header">
          <div>
            <h1> Agrilinks Nigeria Chat </h1>
            <p>
              {" "}
              Connect with farmers, buyers, sellers, agricultural organisations,
              and other members of the Agrilinks Nigeria community.{" "}
            </p>
          </div>
          <a href="/dashboard" className="back-button">
            {" "}
            Dashboard{" "}
          </a>
        </header>
        {truth(data?.Error) ? (
          <>
            <div className="error-message" role="alert">
              {text(data?.Error)}
            </div>
          </>
        ) : null}
        <section className="create-group" aria-labelledby="create-group-title">
          <h2 id="create-group-title"> Create a Group </h2>
          <p>
            {" "}
            Create a community for farmers, buyers, sellers, agricultural
            organisations, associations, or farming discussions.{" "}
          </p>
          <form action="/chat/group/create" method="POST">
            <label htmlFor="group-name" className="sr-only">
              {" "}
              Group name{" "}
            </label>
            <input
              id="group-name"
              type="text"
              name="name"
              placeholder="Example: Maize Farmers Association"
              maxLength="100"
              autoComplete="off"
              required={true}
            />
            <button type="submit"> Create Group </button>
          </form>
        </section>
        <details className="people">
          <summary className="people-toggle">
            <span>
              <strong> People </strong>
              <small> Find someone to start a conversation </small>
            </span>
            <b>
              {text(len(data?.Users))}
              <i aria-hidden="true"> ⌄ </i>
            </b>
          </summary>
          <div className="people-content">
            <form action="/chat" method="GET" className="people-search">
              <label htmlFor="people-search" className="sr-only">
                {" "}
                Search users{" "}
              </label>
              <input
                id="people-search"
                type="search"
                name="q"
                defaultValue={text(data?.Query)}
                placeholder="Find a farmer, buyer, seller, or other user"
                maxLength="100"
                autoComplete="off"
              />
              <button type="submit"> Search </button>
            </form>
            {truth(data?.Users) ? (
              <>
                <div className="people-list">
                  {truth(data?.Users)
                    ? items(data?.Users).map((item1, index1) => (
                        <Fragment key={index1}>
                          <div
                            className="person-row"
                            data-user-id={text(item1?.ID)}
                          >
                            <a
                              className="person-identity"
                              href={"/profile?id=" + text(item1?.ID)}
                            >
                              {truth(item1?.PhotoURL) ? (
                                <>
                                  <img
                                    className="person-avatar"
                                    src={text(item1?.PhotoURL)}
                                    alt={
                                      text(item1?.FullName) + " profile picture"
                                    }
                                    loading="lazy"
                                  />
                                </>
                              ) : (
                                <>
                                  <span
                                    className="person-avatar"
                                    aria-hidden="true"
                                  >
                                    {" "}
                                    👤{" "}
                                  </span>
                                </>
                              )}
                              <span>
                                <strong>{text(item1?.FullName)}</strong>
                                <small>
                                  {text(item1?.Role)}
                                  {truth(item1?.Location) ? (
                                    <> · {text(item1?.Location)}</>
                                  ) : null}
                                </small>
                                <span className="presence-status">
                                  <i aria-hidden="true"></i>
                                  <span> Offline </span>
                                </span>
                              </span>
                            </a>
                            <form action="/chat/private" method="POST">
                              <input
                                type="hidden"
                                name="user_id"
                                defaultValue={text(item1?.ID)}
                              />
                              <button type="submit"> Chat </button>
                            </form>
                          </div>
                        </Fragment>
                      ))
                    : null}
                </div>
              </>
            ) : (
              <>
                <p> No other registered users found. </p>
              </>
            )}
          </div>
        </details>
        <section className="conversations" aria-labelledby="my-chats-title">
          <div className="section-title">
            <h2 id="my-chats-title"> My Chats </h2>
            {truth(data?.UnreadCount > 0) ? (
              <>
                <strong className="unread-summary">
                  {text(data?.UnreadCount)} unread{" "}
                </strong>
              </>
            ) : null}
            <span>{text(len(data?.Conversations))}</span>
          </div>
          {truth(data?.Conversations) ? (
            <>
              <div className="conversation-list">
                {truth(data?.Conversations)
                  ? items(data?.Conversations).map((item2, index2) => (
                      <Fragment key={index2}>
                        <a
                          href={"/chat/view?id=" + text(item2?.ID)}
                          className="conversation-card"
                          data-user-id={text(item2?.OtherUserID)}
                        >
                          <div className="conversation-icon">
                            {truth(item2?.ConversationType === "group") ? (
                              <>
                                <span
                                  className="group-avatar"
                                  aria-hidden="true"
                                >
                                  {" "}
                                  👥{" "}
                                </span>
                              </>
                            ) : (
                              <>
                                {truth(item2?.OtherPhotoURL) ? (
                                  <>
                                    <img
                                      src={text(item2?.OtherPhotoURL)}
                                      alt={
                                        text(item2?.OtherName) +
                                        " profile picture"
                                      }
                                      loading="lazy"
                                    />
                                  </>
                                ) : (
                                  <>
                                    <span
                                      className="group-avatar"
                                      aria-hidden="true"
                                    >
                                      {" "}
                                      👤{" "}
                                    </span>
                                  </>
                                )}
                              </>
                            )}
                          </div>
                          <div className="conversation-info">
                            <h3>
                              {truth(item2?.ConversationType === "private") ? (
                                <>
                                  {truth(item2?.OtherName) ? (
                                    <>{text(item2?.OtherName)}</>
                                  ) : (
                                    <> Private Chat </>
                                  )}
                                </>
                              ) : truth(item2?.Title) ? (
                                <>{text(item2?.Title)}</>
                              ) : (
                                <> Group Chat </>
                              )}
                            </h3>
                            <p>
                              {truth(item2?.ConversationType === "group") ? (
                                <> Group conversation </>
                              ) : (
                                <>
                                  <span className="conversation-presence">
                                    <i aria-hidden="true"></i>
                                    <span> Offline </span>
                                  </span>
                                </>
                              )}
                            </p>
                          </div>
                          <div className="conversation-arrow">
                            {truth(item2?.UnreadCount > 0) ? (
                              <>
                                <span
                                  className="chat-unread"
                                  aria-label={
                                    text(item2?.UnreadCount) +
                                    " unread messages"
                                  }
                                >
                                  {text(item2?.UnreadCount)}
                                </span>
                              </>
                            ) : null}
                            <span aria-hidden="true"> → </span>
                          </div>
                        </a>
                      </Fragment>
                    ))
                  : null}
              </div>
            </>
          ) : (
            <>
              <div className="empty-state">
                <div className="empty-icon" aria-hidden="true">
                  {" "}
                  💬{" "}
                </div>
                <h3> No conversations yet </h3>
                <p>
                  {" "}
                  Create a group or start a private conversation to begin
                  chatting.{" "}
                </p>
              </div>
            </>
          )}
        </section>{" "}
      </main>
    </>
  );
}
