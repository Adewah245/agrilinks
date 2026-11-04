import Navbar from "../components/Navbar.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  money,
  formatDate,
  len,
  requestKey,
} from "../lib/view.js";
export default function CartPage({ data }) {
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
          <a href="/cart" aria-current="page">
            {" "}
            Cart{" "}
          </a>
          <a href="/chat"> Chat </a>
          <a href="/ai-assistant"> AI Assistant </a>
          <a href="/profile"> Profile </a>
          <a href="/notifications"> Notifications </a>
          <a href="/logout"> Logout </a>
        </nav>
      </Navbar>
      <header className="cart-header">
        <div>
          <h1> My Cart </h1>
          <p> Products you have agreed to purchase. </p>
        </div>
        <a href="/marketplace" className="back-button">
          {" "}
          ← Marketplace{" "}
        </a>
      </header>
      {truth(data?.Error) ? (
        <>
          <div className="error-message" role="alert">
            {text(data?.Error)}
          </div>
        </>
      ) : null}
      <main className="cart-container">
        {truth(data?.Items) ? (
          <>
            <div className="cart-items">
              {truth(data?.Items)
                ? items(data?.Items).map((item1, index1) => (
                    <Fragment key={index1}>
                      <article className="cart-item">
                        <div className="cart-image">
                          {truth(item1?.ImageURL) ? (
                            <>
                              <img
                                src={text(item1?.ImageURL)}
                                alt={text(item1?.ProductName)}
                              />
                            </>
                          ) : (
                            <>
                              <div
                                className="no-image"
                                role="img"
                                aria-label="No product image available"
                              >
                                {" "}
                                No Image{" "}
                              </div>
                            </>
                          )}
                        </div>
                        <div className="cart-product">
                          <h2>{text(item1?.ProductName)}</h2>
                          <p>
                            {" "}
                            How much:{" "}
                            <strong>
                              {text(item1?.Quantity)}
                              {text(item1?.Unit)}
                            </strong>
                          </p>
                          <p>
                            {" "}
                            Price for each {text(item1?.Unit)}:{" "}
                            <strong>
                              {" "}
                              ₦{text(money(item1?.PricePerUnit))}
                            </strong>
                          </p>
                          <p className="total-price">
                            {" "}
                            Total:{" "}
                            <strong> ₦{text(money(item1?.TotalPrice))}</strong>
                          </p>
                        </div>
                        <div className="seller-information">
                          <h3> Seller </h3>
                          {truth(item1?.SellerPhotoURL) ? (
                            <>
                              <img
                                className="profile-image"
                                src={text(item1?.SellerPhotoURL)}
                                alt={text(item1?.SellerName)}
                              />
                            </>
                          ) : null}
                          <p>
                            <strong>{text(item1?.SellerName)}</strong>
                          </p>
                          {truth(item1?.SellerEmail) ? (
                            <>
                              <p> Email: {text(item1?.SellerEmail)}</p>
                            </>
                          ) : null}
                          {truth(item1?.SellerPhone) ? (
                            <>
                              <p> Phone: {text(item1?.SellerPhone)}</p>
                            </>
                          ) : null}
                          {truth(item1?.SellerLocation) ? (
                            <>
                              <p> Location: {text(item1?.SellerLocation)}</p>
                            </>
                          ) : null}
                        </div>
                        <div className="buyer-information">
                          <h3> Buyer </h3>
                          {truth(item1?.BuyerPhotoURL) ? (
                            <>
                              <img
                                className="profile-image"
                                src={text(item1?.BuyerPhotoURL)}
                                alt={text(item1?.BuyerName)}
                              />
                            </>
                          ) : null}
                          <p>
                            <strong>{text(item1?.BuyerName)}</strong>
                          </p>
                          {truth(item1?.BuyerEmail) ? (
                            <>
                              <p> Email: {text(item1?.BuyerEmail)}</p>
                            </>
                          ) : null}
                          {truth(item1?.BuyerPhone) ? (
                            <>
                              <p> Phone: {text(item1?.BuyerPhone)}</p>
                            </>
                          ) : null}
                          {truth(item1?.BuyerLocation) ? (
                            <>
                              <p> Location: {text(item1?.BuyerLocation)}</p>
                            </>
                          ) : null}
                        </div>
                        <div className="negotiation-information">
                          <p>
                            {" "}
                            Negotiation ID:{" "}
                            <strong> #{text(item1?.NegotiationID)}</strong>
                          </p>
                          <p>
                            {" "}
                            Agreed price for each {text(item1?.Unit)}:{" "}
                            <strong>
                              {" "}
                              ₦{text(money(item1?.PricePerUnit))}
                            </strong>
                          </p>
                          <p>
                            {" "}
                            Added:{" "}
                            <time
                              dateTime={text(
                                formatDate(
                                  item1?.CreatedAt,
                                  "2006-01-02T15:04:05Z07:00",
                                ),
                              )}
                            >
                              {text(
                                formatDate(
                                  item1?.CreatedAt,
                                  "02 Jan 2006 03:04 PM",
                                ),
                              )}
                            </time>
                          </p>
                        </div>
                        <form
                          method="POST"
                          action="/cart"
                          className="remove-form"
                        >
                          <input
                            type="hidden"
                            name="action"
                            defaultValue="remove"
                          />
                          <input
                            type="hidden"
                            name="cart_id"
                            defaultValue={text(item1?.ID)}
                          />
                          <button
                            type="submit"
                            className="remove-button"
                            aria-label={
                              "Remove " +
                              text(item1?.ProductName) +
                              " from cart"
                            }
                          >
                            {" "}
                            Remove{" "}
                          </button>
                        </form>
                      </article>
                    </Fragment>
                  ))
                : null}
            </div>
            <section className="cart-summary">
              <h2> Cart Summary </h2>
              <p>
                {" "}
                Items in cart: <strong>{text(len(data?.Items))}</strong>
              </p>
              <p>
                {" "}
                Negotiated total: <strong> ₦{text(money(data?.Total))}</strong>
              </p>
              <p> These prices are the prices agreed during negotiation. </p>
              <form method="POST" action="/orders/place">
                <input
                  type="hidden"
                  name="checkout_key"
                  defaultValue={text(requestKey())}
                />
                <button type="submit"> Place Order </button>
              </form>
            </section>
          </>
        ) : (
          <>
            <section className="empty-cart">
              <h2> Your cart is empty </h2>
              <p>
                {" "}
                When you accept a negotiated offer, the product will appear
                here.{" "}
              </p>
              <a href="/marketplace" className="marketplace-button">
                {" "}
                Go to Marketplace{" "}
              </a>
            </section>
          </>
        )}
      </main>
    </>
  );
}
