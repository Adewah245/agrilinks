import Navbar from "../components/Navbar.jsx";
import { Fragment } from "react";
import { truth, items, text, humanNegotiationStatus } from "../lib/view.js";
export default function NegotiationsPage({ data }) {
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
          <a href="/negotiations" aria-current="page">
            {" "}
            Negotiations{" "}
          </a>
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
      <main className="negotiations">
        <section
          className="dashboard-header"
          aria-labelledby="negotiations-title"
        >
          <div className="dashboard-intro">
            <p className="eyebrow"> Agricultural Commerce </p>
            <h1 id="negotiations-title"> My Negotiations </h1>
            <p>
              {" "}
              Price negotiations you have started or received with buyers and
              sellers.{" "}
            </p>
          </div>
        </section>
        <section className="panel" aria-label="My negotiations">
          <div className="table-container">
            <div
              className="table-scroll"
              role="region"
              aria-label="Negotiations table"
              tabIndex="0"
            >
              <table>
                <thead>
                  <tr>
                    <th scope="col"> Product </th>
                    <th scope="col"> With </th>
                    <th scope="col"> Status </th>
                    <th scope="col"> Offers exchanged </th>
                    <th scope="col">
                      <span className="sr-only"> Actions </span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {truth(data?.Negotiations) ? (
                    items(data?.Negotiations).map((item1, index1) => (
                      <Fragment key={index1}>
                        <tr>
                          <td>{text(item1?.ProductName)}</td>
                          <td className="negotiation-person">
                            {truth(item1?.BuyerID === data?.UserID) ? (
                              <>
                                {truth(item1?.SellerPhotoURL) ? (
                                  <>
                                    <img
                                      src={text(item1?.SellerPhotoURL)}
                                      alt=""
                                      loading="lazy"
                                    />
                                  </>
                                ) : (
                                  <>
                                    <span aria-hidden="true"> 👤 </span>
                                  </>
                                )}
                                {text(item1?.SellerName)}
                              </>
                            ) : (
                              <>
                                {truth(item1?.BuyerPhotoURL) ? (
                                  <>
                                    <img
                                      src={text(item1?.BuyerPhotoURL)}
                                      alt=""
                                      loading="lazy"
                                    />
                                  </>
                                ) : (
                                  <>
                                    <span aria-hidden="true"> 👤 </span>
                                  </>
                                )}
                                {text(item1?.BuyerName)}
                              </>
                            )}
                          </td>
                          <td>
                            <span
                              className={"status status-" + text(item1?.Status)}
                            >
                              {text(humanNegotiationStatus(item1?.Status))}
                            </span>
                          </td>
                          <td>
                            {text(item1?.RoundCount)} of{" "}
                            {text(item1?.MaxRounds)}
                          </td>
                          <td>
                            <a
                              className="open-chat-btn"
                              href={"/negotiation?id=" + text(item1?.ID)}
                              aria-label={
                                "Open negotiation for " +
                                text(item1?.ProductName)
                              }
                            >
                              {" "}
                              Open conversation{" "}
                            </a>
                          </td>
                        </tr>
                      </Fragment>
                    ))
                  ) : (
                    <>
                      <tr>
                        <td colSpan="5">
                          <div className="empty-state">
                            <p>
                              {" "}
                              No negotiations yet. Start a negotiation from a
                              Marketplace listing.{" "}
                            </p>
                            <a href="/marketplace" className="primary-btn">
                              {" "}
                              Browse Marketplace{" "}
                            </a>
                          </div>
                        </td>
                      </tr>
                    </>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      </main>
    </>
  );
}
