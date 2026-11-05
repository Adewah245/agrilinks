import { WorkspaceSidebar } from "../components/Workspace.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  money,
  humanOrderStatus,
  formatDate,
  printf,
} from "../lib/view.js";
export default function SalesPage({ data }) {
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/sales">
        <a href="/dashboard"> Dashboard </a>
        <a href="/storage"> Storage </a>
        <a href="/marketplace"> Marketplace </a>
        <a href="/negotiations"> Negotiations </a>
        <a href="/chat"> Chat </a>
        <a href="/sales" aria-current="page">
          {" "}
          Sales{" "}
        </a>
        <a href="/ai-assistant"> AI Assistant </a>
        <a href="/cart"> Cart </a>
        <a href="/profile"> Profile </a>
        <a
          href="/notifications"
          className="notification-link"
          aria-label="Notifications"
        >
          <span className="notification-bell" aria-hidden="true">
            {" "}
            🔔{" "}
          </span>
          <span className="notification-label"> Notifications </span>
        </a>
        <a href="/logout"> Logout </a>
        <a href="/wallet">Wallet</a>
      </WorkspaceSidebar>
      <main className="sales-page">
        <div className="sales-heading">
          <div>
            <p className="eyebrow"> Seller performance </p>
            <h1> Sales </h1>
            <p>
              {" "}
              Review completed orders and revenue from your agricultural
              products.{" "}
            </p>
          </div>
          <a className="back-link" href="/dashboard">
            {" "}
            ← Back to dashboard{" "}
          </a>
        </div>
        <section className="sales-summary" aria-label="Sales summary">
          <div>
            <span> Completed sales </span>
            <strong>{text(data?.Count)}</strong>
            <small> All recorded orders </small>
          </div>
          <div>
            <span> Total revenue </span>
            <strong> ₦{text(money(data?.Total))}</strong>
            <small> Across completed sales </small>
          </div>
        </section>
        <section className="sales-card" aria-labelledby="sales-history-title">
          <h2 id="sales-history-title"> Sales History </h2>
          {truth(data?.Sales) ? (
            <>
              <div className="sales-table">
                <div
                  className="table-scroll"
                  role="region"
                  aria-label="Sales table"
                  tabIndex="0"
                >
                  <table>
                    <thead>
                      <tr>
                        <th scope="col"> Product </th>
                        <th scope="col"> Buyer </th>
                        <th scope="col"> How much </th>
                        <th scope="col"> Total </th>
                        <th scope="col"> Status </th>
                        <th scope="col"> Date </th>
                      </tr>
                    </thead>
                    <tbody>
                      {truth(data?.Sales)
                        ? items(data?.Sales).map((item1, index1) => (
                            <Fragment key={index1}>
                              <tr>
                                <td>{text(item1?.ProductName)}</td>
                                <td>{text(item1?.BuyerName)}</td>
                                <td>
                                  {text(printf("%.0f", item1?.Quantity))}
                                  {text(item1?.ProductUnit)}
                                </td>
                                <td> ₦{text(money(item1?.TotalPrice))}</td>
                                <td>
                                  <span className="sale-status">
                                    {text(humanOrderStatus(item1?.Status))}
                                  </span>
                                </td>
                                <td>
                                  <time dateTime={text(item1?.CreatedAt)}>
                                    {text(
                                      formatDate(
                                        item1?.CreatedAt,
                                        "02 Jan 2006",
                                      ),
                                    )}
                                  </time>
                                </td>
                              </tr>
                            </Fragment>
                          ))
                        : null}
                    </tbody>
                  </table>
                </div>
              </div>
            </>
          ) : (
            <>
              <div className="sales-empty" aria-live="polite">
                <div className="sales-empty-icon" aria-hidden="true">
                  {" "}
                  📦{" "}
                </div>
                <h3> No completed sales yet </h3>
                <p>
                  {" "}
                  List your agricultural products and respond to buyer
                  negotiations to record your first sale.{" "}
                </p>
                <a href="/storage"> Manage Products </a>
              </div>
            </>
          )}
        </section>
      </main>
    </div>
  );
}
