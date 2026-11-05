import { WorkspaceSidebar } from "../components/Workspace.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  money,
  humanOrderStatus,
  formatDate,
} from "../lib/view.js";
export default function OrdersPage({ data }) {
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/orders">
        <a href="/dashboard">Dashboard</a>
        <a href="/marketplace">Marketplace</a>
        <a href="/cart">Cart</a>
        <a href="/orders">Orders</a>
        <a href="/payments">Payments</a>
        <a href="/logout">Logout</a>
        <a href="/wallet">Wallet</a>
      </WorkspaceSidebar>
      <main className="dashboard account-page">
        <p>
          <a href="/dashboard">← Dashboard</a>
        </p>
        <h1>My Orders</h1>
        {truth(data?.Error) ? (
          <>
            <p role="alert">{text(data?.Error)}</p>
          </>
        ) : null}
        {truth(data?.Orders) ? (
          <>
            <section className="panel">
              <div
                className="table-scroll"
                role="region"
                aria-label="Orders table"
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
                      <th>Status</th>
                      <th>Placed</th>
                    </tr>
                  </thead>
                  <tbody>
                    {truth(data?.Orders)
                      ? items(data?.Orders).map((item1, index1) => (
                          <Fragment key={index1}>
                            <tr>
                              <td>
                                #{text(item1?.ID)}
                                <br />
                                <a
                                  href={
                                    "/orders/protection?order_id=" +
                                    text(item1?.ID)
                                  }
                                >
                                  Purchase protection
                                </a>
                              </td>
                              <td>{text(item1?.ProductName)}</td>
                              <td>
                                {text(item1?.Quantity)}
                                {text(item1?.ProductUnit)}
                              </td>
                              <td>{text(item1?.SellerName)}</td>
                              <td>₦{text(money(item1?.TotalPrice))}</td>
                              <td>{text(humanOrderStatus(item1?.Status))}</td>
                              <td>
                                {text(
                                  formatDate(
                                    item1?.CreatedAt,
                                    "02 Jan 2006 03:04 PM",
                                  ),
                                )}
                              </td>
                            </tr>
                          </Fragment>
                        ))
                      : null}
                  </tbody>
                </table>
              </div>
            </section>
            <section className="panel">
              <h2>Review a purchase</h2>
              {truth(data?.Orders)
                ? items(data?.Orders).map((item2, index2) => (
                    <Fragment key={index2}>
                      {truth(item2?.Status === "delivered") ? (
                        <>
                          <form method="POST" action="/reviews">
                            <h3>
                              {text(item2?.ProductName)} — order #
                              {text(item2?.ID)}
                            </h3>
                            <input
                              type="hidden"
                              name="product_id"
                              defaultValue={text(item2?.ProductID)}
                            />
                            <input
                              type="hidden"
                              name="order_id"
                              defaultValue={text(item2?.ID)}
                            />
                            <label>
                              {" "}
                              Rating{" "}
                              <Select name="rating" required={true}>
                                <option value="5">5 — Excellent</option>
                                <option value="4">4 — Good</option>
                                <option value="3">3 — Fair</option>
                                <option value="2">2 — Poor</option>
                                <option value="1">1 — Bad</option>
                              </Select>
                            </label>
                            <label>
                              {" "}
                              Comment{" "}
                              <Textarea name="comment" rows="2"></Textarea>
                            </label>
                            <button type="submit">Submit review</button>
                          </form>
                        </>
                      ) : null}
                    </Fragment>
                  ))
                : null}
            </section>
          </>
        ) : (
          <>
            <section className="panel">
              <p>No orders yet.</p>
              <a href="/marketplace">Browse the marketplace</a>
            </section>
          </>
        )}
        <p>
          <a href="/payments">Continue to payments</a>
        </p>
      </main>
    </div>
  );
}
