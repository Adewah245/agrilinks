import Navbar from "../components/Navbar.jsx";
import { Fragment, useState } from "react";
import {
  truth,
  items,
  text,
  money,
  urlquery,
  requestKey,
} from "../lib/view.js";
export default function MarketplacePage({ data }) {
  const [search, setSearch] = useState(data.SearchTerm || "");
  const products = (data.Products || []).filter((product) =>
    product.Name.toLowerCase().includes(search.toLowerCase()),
  );
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
          {truth(!truth(data?.IsBuyer)) ? (
            <>
              <a href="/storage"> Storage </a>
            </>
          ) : null}
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
      <main className="marketplace">
        <div className="market-header">
          <div>
            <p className="eyebrow"> Agricultural Commerce </p>
            <h1> Marketplace </h1>
            {truth(data?.IsBuyer) ? (
              <>
                <p>
                  {" "}
                  Browse agricultural products from verified sellers, compare
                  available products, place orders, and negotiate prices.{" "}
                </p>
              </>
            ) : (
              <>
                <p>
                  {" "}
                  Discover agricultural products currently listed for sale
                  across Agrilinks Nigeria.{" "}
                </p>
              </>
            )}
          </div>
        </div>
        {truth(data?.Message) ? (
          <>
            <p
              className="message success-message"
              role="status"
              aria-live="polite"
            >
              {text(data?.Message)}
            </p>
          </>
        ) : null}
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
        <form
          method="POST"
          action="/marketplace"
          className="marketplace-search-form"
          role="search"
        >
          <input type="hidden" name="action" defaultValue="search" />
          <label htmlFor="search" className="sr-only">
            {" "}
            Search agricultural products{" "}
          </label>
          <input
            type="search"
            name="search_term"
            id="search"
            placeholder="Search agricultural products..."
            className="search-box"
            autoComplete="off"
            maxLength="100"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <button type="submit" aria-label="Search agricultural products">
            {" "}
            Search{" "}
          </button>
        </form>
        <section
          className="product-grid"
          id="product-grid"
          aria-label="Available agricultural products"
        >
          {truth(data?.Products) ? (
            products.map((item1, index1) => (
              <Fragment key={index1}>
                <article className="product-card">
                  {truth(item1?.PrimaryImageURL) ? (
                    <>
                      <img
                        src={text(item1?.PrimaryImageURL)}
                        alt={text(item1?.Name)}
                        className="product-image"
                        loading="lazy"
                      />
                    </>
                  ) : null}
                  <h3>{text(item1?.Name)}</h3>
                  <p className="price">
                    {" "}
                    ₦{text(money(item1?.Price))} per {text(item1?.Unit)}
                  </p>
                  <p>
                    <span className="seller-identity">
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
                      )}{" "}
                      Seller: {text(item1?.SellerName)}
                    </span>
                  </p>
                  <p>
                    {text(item1?.Quantity)}
                    {text(item1?.Unit)} available{" "}
                  </p>
                  <p className="product-location">
                    <strong> Location: </strong>
                    {truth(item1?.Location) ? (
                      <>
                        <a
                          href={
                            "https://www.google.com/maps/search/?api=1&query=" +
                            text(urlquery(item1?.Location))
                          }
                          target="_blank"
                          rel="noopener noreferrer"
                          aria-label={
                            "View " +
                            text(item1?.Name) +
                            " location on Google Maps"
                          }
                        >
                          {text(item1?.Location)}
                          <span aria-hidden="true"> ↗ </span>
                        </a>
                      </>
                    ) : (
                      <>
                        <span> Location not provided </span>
                      </>
                    )}
                  </p>
                  <a
                    href={"/product?id=" + text(item1?.ID)}
                    className="view-product"
                    aria-label={
                      "View " + text(item1?.Name) + " product details"
                    }
                  >
                    {" "}
                    View Product{" "}
                  </a>
                  {truth(data?.IsBuyer) ? (
                    <>
                      <form
                        method="POST"
                        action="/marketplace"
                        className="buy-form"
                      >
                        <input
                          type="hidden"
                          name="checkout_key"
                          defaultValue={text(requestKey())}
                        />
                        <input
                          type="hidden"
                          name="product_id"
                          defaultValue={text(item1?.ID)}
                        />
                        <label
                          htmlFor={"buy-quantity-" + text(item1?.ID)}
                          className="sr-only"
                        >
                          {" "}
                          How much {text(item1?.Name)} do you want to buy?{" "}
                        </label>
                        <input
                          type="number"
                          id={"buy-quantity-" + text(item1?.ID)}
                          name="quantity"
                          min="0.01"
                          max={text(item1?.Quantity)}
                          step="0.01"
                          placeholder={"Amount in " + text(item1?.Unit)}
                          required={true}
                          inputMode="decimal"
                          aria-label={
                            "How much " +
                            text(item1?.Name) +
                            " do you want to buy?"
                          }
                        />
                        <button type="submit"> Buy now </button>
                        <button
                          type="submit"
                          formAction="/cart"
                          name="action"
                          value="add"
                        >
                          Add to cart
                        </button>
                      </form>
                    </>
                  ) : null}
                  {truth(item1?.OwnerID !== data?.CurrentUserID) ? (
                    <>
                      <form
                        method="POST"
                        action="/negotiation/start"
                        className="negotiate-form"
                      >
                        <input
                          type="hidden"
                          name="product_id"
                          defaultValue={text(item1?.ID)}
                        />
                        <label
                          htmlFor={"negotiation-quantity-" + text(item1?.ID)}
                          className="sr-only"
                        >
                          {" "}
                          How much {text(item1?.Name)} do you want to negotiate
                          for?{" "}
                        </label>
                        <input
                          type="number"
                          id={"negotiation-quantity-" + text(item1?.ID)}
                          name="quantity"
                          min="0.01"
                          max={text(item1?.Quantity)}
                          step="0.01"
                          placeholder={"Amount in " + text(item1?.Unit)}
                          required={true}
                          inputMode="decimal"
                        />
                        <label
                          htmlFor={"offer-price-" + text(item1?.ID)}
                          className="sr-only"
                        >
                          {" "}
                          Your offer for {text(item1?.Name)}
                        </label>
                        <input
                          type="number"
                          id={"offer-price-" + text(item1?.ID)}
                          name="offer_price"
                          min="0.01"
                          step="0.01"
                          placeholder="Your offer ₦"
                          required={true}
                          inputMode="decimal"
                        />
                        <button type="submit"> Negotiate </button>
                      </form>
                    </>
                  ) : null}
                </article>
              </Fragment>
            ))
          ) : (
            <>
              <div className="empty-state">
                <p>
                  {truth(data?.SearchTerm) ? (
                    <>
                      {" "}
                      No available products match “{text(data?.SearchTerm)}
                      ”.{" "}
                    </>
                  ) : (
                    <>
                      {" "}
                      No agricultural products are currently listed for
                      sale.{" "}
                    </>
                  )}
                </p>
                {truth(!truth(data?.IsBuyer)) ? (
                  <>
                    <a href="/storage" className="primary-btn">
                      {" "}
                      Add Product{" "}
                    </a>
                  </>
                ) : null}
              </div>
            </>
          )}
        </section>
      </main>
    </>
  );
}
