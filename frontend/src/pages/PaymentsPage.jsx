import { WorkspaceSidebar } from "../components/Workspace.jsx";
import { Fragment } from "react";
import { truth, items, text, money, humanPaymentStatus } from "../lib/view.js";
export default function PaymentsPage({ data }) {
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/payments">
        <a href="/dashboard">Dashboard</a>
        <a href="/marketplace">Marketplace</a>
        <a href="/cart">Cart</a>
        <a href="/orders">Orders</a>
        <a href="/payments">Payments</a>
        <a href="/wallet">Wallet</a>
        <a href="/logout">Logout</a>
      </WorkspaceSidebar>
      <main className="dashboard account-page">
        <p>
          <a href="/dashboard">← Dashboard</a>
        </p>
        <h1>Payments</h1>
        <p>
          Payments are verified with Flutterwave before an order is confirmed.
        </p>
        {truth(data?.Error) ? (
          <>
            <p role="alert">{text(data?.Error)}</p>
          </>
        ) : null}
        {truth(data?.Notice) ? (
          <>
            <p role="status">{text(data?.Notice)}</p>
          </>
        ) : null}
        <section className="panel">
          <h2>Orders awaiting payment</h2>
          {truth(data?.Orders) ? (
            <>
              <div
                className="table-scroll"
                role="region"
                aria-label="Payments table"
                tabIndex="0"
              >
                <table>
                  <thead>
                    <tr>
                      <th>Order</th>
                      <th>Product</th>
                      <th>How much</th>
                      <th>Seller</th>
                      <th>Total</th>
                      <th>Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {truth(data?.Orders)
                      ? items(data?.Orders).map((item1, index1) => (
                          <Fragment key={index1}>
                            <tr>
                              <td>#{text(item1?.ID)}</td>
                              <td>{text(item1?.ProductName)}</td>
                              <td>
                                {truth(item1?.Items) ? (
                                  items(item1?.Items).map((item2, index2) => (
                                    <Fragment key={index2}>
                                      {text(item2?.Quantity)}
                                      {text(item2?.ProductName)}
                                      <br />
                                    </Fragment>
                                  ))
                                ) : (
                                  <>
                                    {text(item1?.Quantity)}
                                    {text(item1?.ProductUnit)}
                                  </>
                                )}
                              </td>
                              <td>{text(item1?.SellerName)}</td>
                              <td>
                                {text(item1?.Currency)}
                                {text(money(item1?.TotalAmount))}
                              </td>
                              <td>
                                <form method="POST" action="/payments">
                                  <input
                                    type="hidden"
                                    name="order_id"
                                    defaultValue={text(item1?.ID)}
                                  />
                                  <button type="submit">
                                    Pay with Flutterwave
                                  </button>
                                </form>
                                <form method="POST" action="/wallet">
                                  <input
                                    type="hidden"
                                    name="order_id"
                                    defaultValue={text(item1?.ID)}
                                  />
                                  <label>
                                    Transaction PIN{" "}
                                    <input
                                      name="transaction_pin"
                                      type="password"
                                      inputMode="numeric"
                                      pattern="[0-9]{6}"
                                      maxLength="6"
                                      autoComplete="off"
                                      placeholder="6-digit PIN"
                                      required={true}
                                    />
                                  </label>
                                  <small>
                                    Create your PIN in{" "}
                                    <a href="/wallet#wallet-pin-setup">
                                      Wallet
                                    </a>{" "}
                                    before paying from your balance.
                                  </small>
                                  <button
                                    type="submit"
                                    name="action"
                                    value="pay_order"
                                  >
                                    Pay from wallet
                                  </button>
                                </form>
                              </td>
                            </tr>
                          </Fragment>
                        ))
                      : null}
                  </tbody>
                </table>
              </div>
            </>
          ) : (
            <>
              <p>No orders are waiting for payment.</p>
              <a href="/orders">View orders</a>
            </>
          )}
        </section>
        <section className="panel">
          <h2>Payment history</h2>
          {truth(data?.Payments) ? (
            <>
              <div
                className="table-scroll"
                role="region"
                aria-label="Payments table"
                tabIndex="0"
              >
                <table>
                  <thead>
                    <tr>
                      <th>Order</th>
                      <th>Product</th>
                      <th>Amount paid</th>
                      <th>Status</th>
                      <th>Reference</th>
                    </tr>
                  </thead>
                  <tbody>
                    {truth(data?.Payments)
                      ? items(data?.Payments).map((item3, index3) => (
                          <Fragment key={index3}>
                            <tr>
                              <td>#{text(item3?.OrderID)}</td>
                              <td>{text(item3?.ProductName)}</td>
                              <td>
                                {text(item3?.Currency)}
                                {text(item3?.Amount)}
                              </td>
                              <td>
                                {text(humanPaymentStatus(item3?.Status))}
                                {truth(item3?.ReviewRequired) ? (
                                  <> — needs a payment check</>
                                ) : null}
                              </td>
                              <td>{text(item3?.Reference)}</td>
                            </tr>
                          </Fragment>
                        ))
                      : null}
                  </tbody>
                </table>
              </div>
            </>
          ) : (
            <>
              <p>No payment attempts yet.</p>
            </>
          )}
        </section>
      </main>
    </div>
  );
}
