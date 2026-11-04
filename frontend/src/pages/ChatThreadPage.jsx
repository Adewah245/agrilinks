import Navbar from "../components/Navbar.jsx";
import { Fragment, useEffect, useRef } from "react";
import { truth, items, text, formatDate } from "../lib/view.js";
export default function ChatThreadPage({ data }) {
  const messages = useRef(null);
  useEffect(() => {
    if (messages.current)
      messages.current.scrollTop = messages.current.scrollHeight;
  }, []);
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
          <a href="/chat" aria-current="page">
            {" "}
            Chat{" "}
          </a>
          <a href="/storage"> Storage </a>
          <a href="/marketplace"> Marketplace </a>
          <a href="/negotiations"> Negotiations </a>
          <a href="/cart"> Cart </a>
          <a href="/ai-assistant"> AI Assistant </a>
          <a href="/profile"> Profile </a>
          <a href="/notifications"> Notifications </a>
          <a href="/logout"> Logout </a>
        </nav>
      </Navbar>
      <div className="chat-container">
        <header className="chat-header">
          <div className="chat-header-left">
            <a
              href="/chat"
              className="back-button"
              title="Back to chats"
              aria-label="Back to chats"
            >
              {" "}
              ←{" "}
            </a>
            <div className="chat-title">
              <h2 className="chat-heading">
                {truth(data?.Conversation?.ConversationType === "group") ? (
                  <>{text(data?.Conversation?.Title)}</>
                ) : (
                  <>
                    {truth(data?.Members)
                      ? items(data?.Members).map((item1, index1) => (
                          <Fragment key={index1}>
                            {truth(item1?.UserID !== data?.User?.ID) ? (
                              <>
                                <a
                                  className="chat-profile-link"
                                  href={"/profile?id=" + text(item1?.UserID)}
                                >
                                  {truth(item1?.PhotoURL) ? (
                                    <>
                                      <img
                                        className="thread-avatar"
                                        src={text(item1?.PhotoURL)}
                                        alt={
                                          text(item1?.FullName) +
                                          " profile picture"
                                        }
                                      />
                                    </>
                                  ) : (
                                    <>
                                      <span
                                        className="thread-avatar"
                                        role="img"
                                        aria-label="Profile picture unavailable"
                                      >
                                        {" "}
                                        👤{" "}
                                      </span>
                                    </>
                                  )}
                                  {text(item1?.FullName)}
                                </a>
                              </>
                            ) : null}
                          </Fragment>
                        ))
                      : null}
                  </>
                )}
              </h2>
              <span className="chat-type">
                {truth(data?.Conversation?.ConversationType === "group") ? (
                  <> Group conversation </>
                ) : (
                  <> Private conversation </>
                )}
                <span className="chat-status">
                  <i aria-hidden="true"></i> Active conversation{" "}
                </span>
              </span>
            </div>
          </div>
        </header>
        {truth(data?.Conversation?.ConversationType === "group") ? (
          <>
            <section className="members-section" aria-label="Group members">
              <div className="members-title"> Group Members </div>
              <div className="members">
                {truth(data?.Members)
                  ? items(data?.Members).map((item2, index2) => (
                      <Fragment key={index2}>
                        <a
                          className="member"
                          href={"/profile?id=" + text(item2?.UserID)}
                        >
                          {truth(item2?.PhotoURL) ? (
                            <>
                              <img
                                src={text(item2?.PhotoURL)}
                                alt={text(item2?.FullName) + " profile picture"}
                              />
                            </>
                          ) : (
                            <>
                              <span
                                className="member-avatar"
                                role="img"
                                aria-label="Profile picture unavailable"
                              >
                                {" "}
                                👤{" "}
                              </span>
                            </>
                          )}
                          <span>{text(item2?.FullName)}</span>
                        </a>
                      </Fragment>
                    ))
                  : null}
              </div>
            </section>
          </>
        ) : null}
        <main
          className="messages"
          id="messages"
          ref={messages}
          aria-label="Conversation messages"
        >
          {truth(data?.Messages) ? (
            <>
              {truth(data?.Messages)
                ? items(data?.Messages).map((item3, index3) => (
                    <Fragment key={index3}>
                      <div
                        className={
                          "message\n                    " +
                          (truth(item3?.SenderID === data?.User?.ID)
                            ? "\n                        mine\n                    "
                            : "\n                        theirs\n                    ")
                        }
                      >
                        {truth(item3?.SenderID !== data?.User?.ID) ? (
                          <>
                            {truth(data?.Members)
                              ? items(data?.Members).map((item4, index4) => (
                                  <Fragment key={index4}>
                                    {truth(
                                      item4?.UserID === item3?.SenderID,
                                    ) ? (
                                      <>
                                        <a
                                          className="sender"
                                          href={
                                            "/profile?id=" + text(item4?.UserID)
                                          }
                                        >
                                          {truth(item4?.PhotoURL) ? (
                                            <>
                                              <img
                                                src={text(item4?.PhotoURL)}
                                                alt={
                                                  text(item4?.FullName) +
                                                  " profile picture"
                                                }
                                              />
                                            </>
                                          ) : (
                                            <>
                                              <span
                                                role="img"
                                                aria-label="Profile picture unavailable"
                                              >
                                                {" "}
                                                👤{" "}
                                              </span>
                                            </>
                                          )}
                                          {text(item4?.FullName)}
                                        </a>
                                      </>
                                    ) : null}
                                  </Fragment>
                                ))
                              : null}
                          </>
                        ) : null}
                        <div className="message-bubble">
                          {text(item3?.Message)}
                        </div>
                        <div className="message-time">
                          <time
                            className="local-time"
                            dateTime={text(
                              formatDate(
                                item3?.CreatedAt,
                                "2006-01-02T15:04:05Z07:00",
                              ),
                            )}
                          >
                            {text(
                              formatDate(item3?.CreatedAt, "02 Jan 2006 15:04"),
                            )}
                          </time>
                          {truth(item3?.SenderID === data?.User?.ID) ? (
                            <>
                              <span className="message-receipt">
                                {truth(item3?.Seen) ? (
                                  <> ✓✓ Seen </>
                                ) : (
                                  <> ✓ Sent </>
                                )}
                              </span>
                            </>
                          ) : null}
                        </div>
                      </div>
                    </Fragment>
                  ))
                : null}
            </>
          ) : (
            <>
              <div className="empty-chat">
                <h3> No messages yet </h3>
                <p> Start the conversation. </p>
              </div>
            </>
          )}
        </main>
        <form className="message-form" method="POST" action="/chat/send">
          <input
            type="hidden"
            name="conversation_id"
            defaultValue={text(data?.Conversation?.ID)}
          />
          <label htmlFor="message-input" className="visually-hidden">
            {" "}
            Message{" "}
          </label>
          <input
            type="text"
            id="message-input"
            name="message"
            className="message-input"
            placeholder="Type a message..."
            maxLength="5000"
            required={true}
            autoComplete="off"
            aria-label="Message"
          />
          <button
            type="submit"
            className="send-button"
            aria-label="Send message"
          >
            <span> Send </span>
          </button>
        </form>
        <div
          id="typing-status"
          className="typing-status"
          aria-live="polite"
        ></div>
      </div>
    </>
  );
}
