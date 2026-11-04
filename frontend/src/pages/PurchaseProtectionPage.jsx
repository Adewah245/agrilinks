import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text, humanStatus, formatDate } from "../lib/view.js";
export default function PurchaseProtectionPage({ data }) {
  return (
    <>
      <main className="finance-page">
        <nav>
          <a href="/wallet">Wallet</a> · <a href="/orders">Orders</a> ·{" "}
          <a href="/sales">Sales</a>
          {truth(data?.CanReview) ? (
            <>
              {" "}
              · <a href="/admin/finance">Finance</a>
            </>
          ) : null}
        </nav>
        <p className="finance-eyebrow">Agrilinks Nigeria purchase protection</p>
        <h1>Order #{text(data?.Escrow?.OrderID)}</h1>
        <p>{text(data?.Escrow?.Products)}</p>
        {truth(data?.Error) ? (
          <>
            <p className="finance-alert" role="alert">
              {text(data?.Error)}
            </p>
          </>
        ) : null}
        {truth(data?.Notice) ? (
          <>
            <p role="status">{text(data?.Notice)}</p>
          </>
        ) : null}
        <section className="finance-card">
          <h2>{text(humanStatus(data?.Escrow?.State))}</h2>
          <div className="finance-summary">
            <div>
              <span>Protected payment</span>
              <strong>
                {text(data?.Escrow?.Currency)}
                {text(data?.Escrow?.Amount)}
              </strong>
            </div>
            <div>
              <span>Marketplace commission on release</span>
              <strong>
                {text(data?.Escrow?.Currency)}
                {text(data?.Escrow?.Commission)}
              </strong>
            </div>
            <div>
              <span>Seller receives on approval</span>
              <strong>
                {text(data?.Escrow?.Currency)}
                {text(data?.Escrow?.SellerCredit)}
              </strong>
            </div>
          </div>
          <p>
            Buyer: {text(data?.Escrow?.BuyerName)} · Seller:{" "}
            {text(data?.Escrow?.SellerName)}
          </p>
          <p>
            Reference:{" "}
            <span className="finance-reference">
              {text(data?.Escrow?.PaymentReference)}
            </span>
          </p>
          <p>Held funds cannot be withdrawn by the seller.</p>
        </section>
        <section className="finance-card">
          <h2>Order agreement</h2>
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Quantity</th>
                  <th>Agreed unit price</th>
                  <th>Total</th>
                </tr>
              </thead>
              <tbody>
                {truth(data?.Items)
                  ? items(data?.Items).map((item1, index1) => (
                      <Fragment key={index1}>
                        <tr>
                          <td>{text(item1?.Product)}</td>
                          <td>{text(item1?.Quantity)}</td>
                          <td>{text(item1?.UnitPrice)}</td>
                          <td>{text(item1?.Total)}</td>
                        </tr>
                      </Fragment>
                    ))
                  : null}
              </tbody>
            </table>
          </div>
        </section>
        <section className="finance-card">
          <h2>Delivery and release checks</h2>
          <ul>
            <li>Payment held: {text(data?.Escrow?.Amount)}</li>
            <li>
              Delivery:{" "}
              {truth(data?.Escrow?.DeliveredAt?.Valid) ? (
                <>
                  {text(
                    formatDate(
                      data?.Escrow?.DeliveredAt?.Time,
                      "02 Jan 2006 15:04 UTC",
                    ),
                  )}
                </>
              ) : (
                <>Awaiting delivery</>
              )}
            </li>
            <li>
              Buyer confirmation:{" "}
              {truth(data?.Escrow?.BuyerConfirmedAt?.Valid) ? (
                <>
                  {text(
                    formatDate(
                      data?.Escrow?.BuyerConfirmedAt?.Time,
                      "02 Jan 2006 15:04 UTC",
                    ),
                  )}
                </>
              ) : (
                <>Awaiting buyer confirmation</>
              )}
            </li>
            <li>
              Administrator approval:{" "}
              {truth(data?.Escrow?.DecidedAt?.Valid) ? (
                <>{text(data?.Escrow?.DecisionReason)}</>
              ) : (
                <>Awaiting review</>
              )}
            </li>
          </ul>
          {truth(data?.Escrow?.DeliveryEvidence) ? (
            <>
              <p>Delivery evidence: {text(data?.Escrow?.DeliveryEvidence)}</p>
            </>
          ) : null}
          {truth(data?.Escrow?.DisputeReason) ? (
            <>
              <p className="finance-alert">
                Disputed: {text(data?.Escrow?.DisputeReason)}. Funds remain
                frozen.
              </p>
            </>
          ) : null}
          {truth(
            truth(data?.Escrow?.State === "held") ||
              truth(data?.Escrow?.State === "disputed"),
          ) ? (
            <>
              <form method="post" className="finance-form">
                <input
                  type="hidden"
                  name="csrf"
                  defaultValue={text(data?.CSRF)}
                />
                <input
                  type="hidden"
                  name="order_id"
                  defaultValue={text(data?.Escrow?.OrderID)}
                />
                <label>
                  Action
                  <Select name="action" required={true}>
                    {truth(data?.IsSeller) ? (
                      <>
                        <option value="seller_delivered">
                          Record delivery
                        </option>
                        <option value="dispute">Open a dispute</option>
                      </>
                    ) : null}
                    {truth(data?.IsBuyer) ? (
                      <>
                        {truth(data?.Escrow?.DeliveredAt?.Valid) ? (
                          <>
                            <option value="confirm_delivery">
                              Confirm I received this order
                            </option>
                          </>
                        ) : null}
                        <option value="dispute">Open a dispute</option>
                      </>
                    ) : null}
                    {truth(data?.CanReview) ? (
                      <>
                        {truth(data?.Escrow?.Eligible) ? (
                          <>
                            <option value="release">
                              Approve release to seller
                            </option>
                          </>
                        ) : null}
                        {truth(data?.Escrow?.State === "disputed") ? (
                          <>
                            <option value="resolve_dispute">
                              Resolve dispute and request fresh buyer
                              confirmation
                            </option>
                          </>
                        ) : null}
                        <option value="refund">
                          Refund protected funds to buyer's wallet
                        </option>
                      </>
                    ) : null}
                  </Select>
                </label>
                <label>
                  Reason or delivery evidence
                  <Textarea
                    name="reason"
                    minLength="5"
                    maxLength="2000"
                    required={true}
                  ></Textarea>
                </label>
                <button type="submit">Record action</button>
              </form>
            </>
          ) : null}
        </section>
        <section className="finance-card">
          <h2>Decision history</h2>
          {truth(data?.Events)
            ? items(data?.Events).map((item2, index2) => (
                <Fragment key={index2}>
                  <article className="finance-event">
                    <strong>{text(humanStatus(item2?.Action))}</strong>
                    <p>{text(item2?.Reason)}</p>
                    <small>
                      {text(item2?.ActorName)}
                      {truth(item2?.ActorID?.Valid) ? (
                        <> (#{text(item2?.ActorID?.Int64)})</>
                      ) : null}{" "}
                      ·{" "}
                      {text(
                        formatDate(item2?.CreatedAt, "02 Jan 2006 15:04 UTC"),
                      )}
                    </small>
                  </article>
                </Fragment>
              ))
            : null}
        </section>
      </main>
    </>
  );
}
