import Navbar from "../components/Navbar.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  money,
  humanNegotiationStatus,
  humanOfferStatus,
  formatDate,
} from "../lib/view.js";
export default function NegotiationPage({ data }) {
  return (
    <>
      <Navbar className="navbar">
        <a
          className="logo"
          href="/dashboard"
          aria-label="Agrilinks Nigeria dashboard"
        >
          <img
            src="/static/assets/logo/agrilinks-nigeria-logo.svg"
            alt="Agrilinks Nigeria logo"
          />
        </a>
        <nav aria-label="Main navigation">
          <a href="/dashboard"> Dashboard </a>
          <a href="/storage"> Storage </a>
          <a href="/marketplace"> Marketplace </a>
          <a href="/negotiations"> Negotiations </a>
          <a href="/chat"> Chat </a>
          <a href="/cart" aria-label="Shopping cart">
            {" "}
            🛒{" "}
          </a>
          <a href="/ai-assistant"> AI Assistant </a>
          <a href="/profile"> Profile </a>
          <a href="/notifications"> Notifications </a>
          <a href="/logout"> Logout </a>
        </nav>
      </Navbar>
      <main className="negotiation-thread">
        <div className="thread-header">
          <div>
            <h1>{text(data?.Negotiation?.ProductName)}</h1>
            <p>
              {" "}
              Status:{" "}
              <span
                className={"status status-" + text(data?.Negotiation?.Status)}
              >
                {text(humanNegotiationStatus(data?.Negotiation?.Status))}
              </span>{" "}
              • Offers used: {text(data?.Negotiation?.RoundCount)} of{" "}
              {text(data?.Negotiation?.MaxRounds)} • How much:{" "}
              {text(data?.Negotiation?.Quantity)}
              {text(data?.Negotiation?.ProductUnit)}
            </p>
          </div>
          <div
            className={
              "time-left " +
              (truth(data?.TimeLeft === "Expired") ? "expired" : "")
            }
            aria-label="Negotiation time remaining"
          >
            {" "}
            ⏱ {text(data?.TimeLeft)}
          </div>
        </div>
        {truth(data?.Error) ? (
          <>
            <p
              className="message error-message"
              role="alert"
              aria-live="assertive"
            >
              {text(data?.Error)}
            </p>
          </>
        ) : null}
        <div className="chat-thread" aria-label="Negotiation messages">
          {truth(data?.Messages) ? (
            items(data?.Messages).map((item1, index1) => (
              <Fragment key={index1}>
                <div
                  className={
                    "chat-bubble " +
                    (truth(item1?.SenderID === data?.UserID)
                      ? "mine"
                      : "theirs")
                  }
                >
                  {truth(item1?.MessageType === "offer") ? (
                    <>
                      <p className="offer">
                        {" "}
                        ₦{text(money(item1?.OfferPrice))}
                      </p>
                      <span
                        className={
                          "offer-status offer-status-" +
                          text(item1?.OfferStatus)
                        }
                      >
                        {text(humanOfferStatus(item1?.OfferStatus))}
                      </span>
                    </>
                  ) : null}
                  {truth(item1?.Message) ? (
                    <>
                      <p className="text">{text(item1?.Message)}</p>
                    </>
                  ) : null}
                  <small>
                    {text(formatDate(item1?.CreatedAt, "02 Jan, 3:04 PM"))}
                  </small>
                  {truth(
                    truth(data?.Negotiation?.Status === "open") &&
                      truth(item1?.MessageType === "offer") &&
                      truth(item1?.OfferStatus === "pending") &&
                      truth(item1?.SenderID !== data?.UserID),
                  ) ? (
                    <>
                      <div className="offer-actions">
                        <form
                          method="POST"
                          action={
                            "/negotiation?id=" + text(data?.Negotiation?.ID)
                          }
                        >
                          <input
                            type="hidden"
                            name="action"
                            defaultValue="accept"
                          />
                          <input
                            type="hidden"
                            name="offer_id"
                            defaultValue={text(item1?.ID)}
                          />
                          <button type="submit" className="accept-btn">
                            {" "}
                            Accept Offer{" "}
                          </button>
                        </form>
                        <form
                          method="POST"
                          action={
                            "/negotiation?id=" + text(data?.Negotiation?.ID)
                          }
                        >
                          <input
                            type="hidden"
                            name="action"
                            defaultValue="reject"
                          />
                          <input
                            type="hidden"
                            name="offer_id"
                            defaultValue={text(item1?.ID)}
                          />
                          <button type="submit" className="reject-btn">
                            {" "}
                            Reject Offer{" "}
                          </button>
                        </form>
                      </div>
                    </>
                  ) : null}
                </div>
              </Fragment>
            ))
          ) : (
            <>
              <p className="empty-state"> No negotiation messages yet. </p>
            </>
          )}
        </div>
        {truth(data?.Negotiation?.Status === "open") ? (
          <>
            <form
              method="POST"
              action={"/negotiation?id=" + text(data?.Negotiation?.ID)}
              className="chat-form"
            >
              <input type="hidden" name="action" defaultValue="chat" />
              <label htmlFor="negotiation-message" className="sr-only">
                {" "}
                Write a negotiation message{" "}
              </label>
              <input
                type="text"
                id="negotiation-message"
                name="message"
                placeholder="Write a message..."
                maxLength="2000"
                autoComplete="off"
                required={true}
              />
              <button type="submit"> Send Message </button>
            </form>
            {truth(data?.TimeLeft === "Expired") ? (
              <>
                <p className="message error-message" role="alert">
                  {" "}
                  This negotiation has expired. Start a new negotiation from the
                  Marketplace if you are still interested.{" "}
                </p>
              </>
            ) : truth(
                data?.Negotiation?.RoundCount >= data?.Negotiation?.MaxRounds,
              ) ? (
              <>
                <p className="message warning-message" role="status">
                  {" "}
                  You have reached the maximum number of negotiation rounds.
                  Accept the latest offer or reject it to end the
                  negotiation.{" "}
                </p>
              </>
            ) : (
              <>
                <form
                  method="POST"
                  action={"/negotiation?id=" + text(data?.Negotiation?.ID)}
                  className="offer-form"
                >
                  <input type="hidden" name="action" defaultValue="offer" />
                  <label htmlFor="offer-price" className="sr-only">
                    {" "}
                    Your offer{" "}
                  </label>
                  <input
                    type="number"
                    id="offer-price"
                    name="offer_price"
                    placeholder="Your offer (₦)"
                    min="0.01"
                    step="0.01"
                    inputMode="decimal"
                    required={true}
                  />
                  <label htmlFor="offer-message" className="sr-only">
                    {" "}
                    Optional message with your offer{" "}
                  </label>
                  <input
                    type="text"
                    id="offer-message"
                    name="message"
                    placeholder="Message (optional)"
                    maxLength="2000"
                    autoComplete="off"
                  />
                  <button type="submit"> Send Offer </button>
                </form>
              </>
            )}
            <form
              method="POST"
              action={"/negotiation?id=" + text(data?.Negotiation?.ID)}
              className="end-form"
            >
              <input type="hidden" name="action" defaultValue="end" />
              <button type="submit" className="reject-btn">
                {" "}
                End Negotiation{" "}
              </button>
            </form>
          </>
        ) : (
          <>
            <p className="message" role="status">
              <strong>
                {" "}
                This negotiation is{" "}
                {text(humanNegotiationStatus(data?.Negotiation?.Status))}.{" "}
              </strong>
            </p>
          </>
        )}
      </main>
    </>
  );
}
