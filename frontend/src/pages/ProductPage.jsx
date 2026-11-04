import Navbar from "../components/Navbar.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  money,
  money0,
  humanBadge,
  formatDate,
  urlquery,
} from "../lib/view.js";
import mountProduct from "../behaviors/product.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function ProductPage({ data }) {
  usePageBehaviors([mountProduct], data);
  return (
    <>
      <Navbar className="navbar">
        <a
          className="logo"
          href="/dashboard"
          aria-label="Agrilinks Nigeria Dashboard"
        >
          <img
            src="/static/assets/logo/agrilinks-nigeria-logo.svg?v=20260904"
            alt="Agrilinks Nigeria"
          />
        </a>
        <nav aria-label="Main navigation">
          <a href="/dashboard"> Dashboard </a>
          <a href="/storage"> Storage </a>
          <a href="/marketplace" aria-current="page">
            {" "}
            Marketplace{" "}
          </a>
          <a href="/negotiations"> Negotiations </a>
          <a href="/chat"> Chat </a>
          <a href="/cart"> Cart </a>
          <a href="/ai-assistant"> AI Assistant </a>
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
          <a href="/logout"> Logout </a>
        </nav>
      </Navbar>
      <main className="product-page">
        <section className="product-card" aria-labelledby="product-name">
          <h1 id="product-name" className="product-name">
            {text(data?.Product?.Name)}
          </h1>
          {truth(data?.Product?.Description) ? (
            <>
              <p className="product-description">
                {text(data?.Product?.Description)}
              </p>
            </>
          ) : null}
          <div className="product-gallery" aria-label="Product images">
            {truth(data?.Images) ? (
              <>
                {truth(data?.Images)
                  ? items(data?.Images).map((item1, index1) => (
                      <Fragment key={index1}>
                        <div className="product-gallery-item">
                          <img
                            className="product-image"
                            src={text(item1?.ImageURL)}
                            alt={text(data?.Product?.Name)}
                            loading="lazy"
                          />
                        </div>
                      </Fragment>
                    ))
                  : null}
              </>
            ) : (
              <>
                {truth(data?.Product?.Images) ? (
                  <>
                    {truth(data?.Product?.Images)
                      ? items(data?.Product?.Images).map((item2, index2) => (
                          <Fragment key={index2}>
                            <div className="product-gallery-item">
                              <img
                                className="product-image"
                                src={text(item2?.ImageURL)}
                                alt={text(data?.Product?.Name)}
                                loading="lazy"
                              />
                            </div>
                          </Fragment>
                        ))
                      : null}
                  </>
                ) : (
                  <>
                    <div className="product-no-image">
                      <p> No product image available. </p>
                    </div>
                  </>
                )}
              </>
            )}
          </div>
          <p className="product-price">
            {" "}
            ₦{text(money0(data?.Product?.Price))} for each{" "}
            {text(data?.Product?.Unit)}
          </p>
          <div className="product-details">
            <div className="product-detail-box">
              <span className="product-detail-label"> Category </span>
              <span className="product-detail-value">
                {text(data?.Product?.Category)}
              </span>
            </div>
            <div className="product-detail-box">
              <span className="product-detail-label">
                {" "}
                How much is available{" "}
              </span>
              <span className="product-detail-value">
                {text(data?.Product?.Quantity)}
                {text(data?.Product?.Unit)}
              </span>
            </div>
            <div className="product-detail-box">
              <span className="product-detail-label">
                {" "}
                Price for each {text(data?.Product?.Unit)}
              </span>
              <span className="product-detail-value">
                {" "}
                ₦{text(money0(data?.Product?.Price))} for each{" "}
                {text(data?.Product?.Unit)}
              </span>
            </div>
            <div className="product-detail-box">
              <span className="product-detail-label"> Location </span>
              <span className="product-detail-value">
                {truth(data?.Product?.Location) ? (
                  <>
                    <a
                      className="map-location-link"
                      href={
                        "https://www.google.com/maps/search/?api=1&query=" +
                        text(urlquery(data?.Product?.Location))
                      }
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {text(data?.Product?.Location)}
                      <span> ↗ View on map </span>
                    </a>
                  </>
                ) : (
                  <> Location not provided </>
                )}
              </span>
            </div>
            <div className="product-detail-box">
              <span className="product-detail-label"> Seller </span>
              <span className="product-detail-value seller-product">
                {truth(data?.Product?.SellerPhotoURL) ? (
                  <>
                    <img
                      src={text(data?.Product?.SellerPhotoURL)}
                      alt={text(data?.Product?.SellerName)}
                      className="seller-product-photo"
                      loading="lazy"
                    />
                  </>
                ) : (
                  <>
                    <span className="seller-placeholder" aria-hidden="true">
                      {" "}
                      👤{" "}
                    </span>
                  </>
                )}
                <span>
                  {truth(data?.Product?.SellerName) ? (
                    <>{text(data?.Product?.SellerName)}</>
                  ) : (
                    <> Seller </>
                  )}
                </span>
                {truth(data?.SellerTrust?.ReviewCount > 0) ? (
                  <>
                    <span className="seller-trust-line">
                      {text(money(data?.SellerTrust?.AverageRating))}/5 from{" "}
                      {text(data?.SellerTrust?.ReviewCount)} reviews{" "}
                    </span>
                  </>
                ) : null}
                {truth(data?.SellerTrust?.Badges)
                  ? items(data?.SellerTrust?.Badges).map((item3, index3) => (
                      <Fragment key={index3}>
                        <span className="trust-badge">
                          {text(humanBadge(item3?.BadgeType))}
                        </span>
                      </Fragment>
                    ))
                  : null}
              </span>
            </div>
          </div>
          <div className="product-status-section">
            {truth(data?.Product?.Status === "active") ? (
              <>
                {truth(data?.Product?.Quantity > 0.0) ? (
                  <>
                    <span className="product-status available">
                      {" "}
                      Available for Sale{" "}
                    </span>
                  </>
                ) : (
                  <>
                    <span className="product-status not-available">
                      {" "}
                      Sold Out{" "}
                    </span>
                  </>
                )}
              </>
            ) : truth(data?.Product?.Status === "sold_out") ? (
              <>
                <span className="product-status not-available"> Sold Out </span>
              </>
            ) : (
              <>
                <span className="product-status not-available">
                  {" "}
                  Not Available{" "}
                </span>
              </>
            )}
          </div>
          <div className="product-button-group">
            <a className="product-button secondary" href="/marketplace">
              {" "}
              ← Back to Marketplace{" "}
            </a>
          </div>
        </section>
        <section id="reviews" className="product-detail-box">
          <h2>Product Reviews</h2>
          {truth(data?.ProductTrust?.ReviewCount > 0) ? (
            <>
              <p className="review-summary">
                <strong>
                  {text(money(data?.ProductTrust?.AverageRating))} out of 5
                </strong>{" "}
                from {text(data?.ProductTrust?.ReviewCount)} buyer reviews.{" "}
              </p>
            </>
          ) : null}
          {truth(data?.Reviews) ? (
            <>
              <ul>
                {truth(data?.Reviews)
                  ? items(data?.Reviews).map((item4, index4) => (
                      <Fragment key={index4}>
                        <li>
                          <strong>{text(item4?.Rating)}/5</strong>
                          {truth(item4?.ReviewerName) ? (
                            <>— {text(item4?.ReviewerName)}</>
                          ) : null}
                          {truth(item4?.Comment) ? (
                            <>— {text(item4?.Comment)}</>
                          ) : null}
                          <small>
                            {text(formatDate(item4?.CreatedAt, "02 Jan 2006"))}
                          </small>
                        </li>
                      </Fragment>
                    ))
                  : null}
              </ul>
            </>
          ) : (
            <>
              <p>No reviews yet.</p>
            </>
          )}
        </section>
      </main>
    </>
  );
}
