import Navbar from "../components/Navbar.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment, useState } from "react";
import {
  truth,
  items,
  text,
  money,
  hasCoordinates,
  formatDate,
} from "../lib/view.js";
import useStorageEditor from "../hooks/useStorageEditor.js";
export default function StoragePage({ data }) {
  const editor = useStorageEditor(data.EditProduct);
  const editProduct = editor.product;
  const [search, setSearch] = useState("");
  const products = (data.Products || []).filter((product) =>
    product.Name.toLowerCase().includes(search.toLowerCase()),
  );
  return (
    <>
      <Navbar className="navbar">
        <div className="navbar-container">
          <a
            className="navbar-brand"
            href="/dashboard"
            aria-label="Agrilinks Nigeria Dashboard"
          >
            <img
              src="/static/assets/logo/agrilinks-nigeria-logo.svg?v=20260904"
              alt="Agrilinks Nigeria logo"
              className="navbar-logo"
            />
            <span className="navbar-title"> Agrilinks Nigeria </span>
          </a>
          <nav className="navbar-menu" aria-label="Main navigation">
            <a href="/dashboard" className="nav-link">
              {" "}
              Dashboard{" "}
            </a>
            <a href="/storage" className="nav-link" aria-current="page">
              {" "}
              Storage{" "}
            </a>
            <a href="/marketplace" className="nav-link">
              {" "}
              Marketplace{" "}
            </a>
            <a href="/negotiations" className="nav-link">
              {" "}
              Negotiations{" "}
            </a>
            <a href="/chat" className="nav-link">
              {" "}
              Chat{" "}
            </a>
            <a href="/cart" className="nav-link">
              {" "}
              Cart{" "}
            </a>
            <a href="/ai-assistant" className="nav-link">
              {" "}
              AI Assistant{" "}
            </a>
            <a href="/profile" className="nav-link">
              {" "}
              Profile{" "}
            </a>
            <a
              href="/notifications"
              className="nav-link notification-link"
              aria-label="Notifications"
            >
              <span className="notification-bell" aria-hidden="true">
                {" "}
                🔔{" "}
              </span>
              <span className="notification-label"> Notifications </span>
            </a>
            <a href="/logout" className="nav-link logout-link">
              {" "}
              Logout{" "}
            </a>
          </nav>
        </div>
      </Navbar>
      <main className="storage">
        <section className="storage-header" aria-labelledby="storage-title">
          <div>
            <h1 id="storage-title"> My products </h1>
            <p>
              {" "}
              Manage your agricultural products and keep track of what you have,
              and choose what to list on the Marketplace.{" "}
            </p>
          </div>
          <button
            onClick={editor.add}
            id="add-btn"
            type="button"
            className="btn btn-primary"
          >
            {" "}
            Add Product{" "}
          </button>
        </section>
        {truth(data?.Error) ? (
          <>
            <p className="message error-message" role="alert">
              {text(data?.Error)}
            </p>
          </>
        ) : null}
        <section className="storage-search" aria-label="Search stored products">
          <label htmlFor="search" className="sr-only">
            {" "}
            Search products{" "}
          </label>
          <input
            type="search"
            id="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search products..."
            autoComplete="off"
            aria-label="Search products"
          />
        </section>
        <section className="storage-table-wrapper" aria-label="Stored products">
          <div
            className="table-scroll"
            role="region"
            aria-label="My products table"
            tabIndex="0"
          >
            <table>
              <thead>
                <tr>
                  <th scope="col">Photo</th>
                  <th scope="col">Product</th>
                  <th scope="col">Category</th>
                  <th scope="col">Available amount</th>
                  <th scope="col">Sold by</th>
                  <th scope="col">Location</th>
                  <th scope="col">Price</th>
                  <th scope="col">Marketplace</th>
                  <th scope="col">Date Stored</th>
                  <th scope="col">Actions</th>
                </tr>
              </thead>
              <tbody id="storage-body">
                {truth(data?.Products) ? (
                  products.map((item1, index1) => (
                    <Fragment key={index1}>
                      <tr data-product-name={text(item1?.Name)}>
                        <td>
                          {truth(item1?.Images) ? (
                            <>
                              {truth(item1?.Images)
                                ? items(item1?.Images).map((item2, index2) => (
                                    <Fragment key={index2}>
                                      {truth(item2?.IsPrimary) ? (
                                        <>
                                          <img
                                            src={text(item2?.ImageURL)}
                                            alt={
                                              text(item1?.Name) +
                                              " product photo"
                                            }
                                            className="thumb"
                                            loading="lazy"
                                          />
                                        </>
                                      ) : null}
                                    </Fragment>
                                  ))
                                : null}
                            </>
                          ) : (
                            <>
                              <span className="no-image"> — </span>
                            </>
                          )}
                        </td>
                        <td>
                          <strong>{text(item1?.Name)}</strong>
                        </td>
                        <td>
                          {truth(item1?.Category) ? (
                            <>{text(item1?.Category)}</>
                          ) : (
                            <> — </>
                          )}
                        </td>
                        <td>{text(item1?.Quantity)}</td>
                        <td>{text(item1?.Unit)}</td>
                        <td>
                          {truth(item1?.Location) ? (
                            <>{text(item1?.Location)}</>
                          ) : (
                            <> — </>
                          )}
                        </td>
                        <td>
                          {" "}
                          ₦{text(money(item1?.Price))} for each{" "}
                          {text(item1?.Unit)}
                        </td>
                        <td>
                          {truth(item1?.Status === "active") ? (
                            <>
                              <span className="status status-active">
                                {" "}
                                Listed{" "}
                              </span>
                            </>
                          ) : truth(item1?.Status === "sold_out") ? (
                            <>
                              <span className="status status-sold-out">
                                {" "}
                                Sold Out{" "}
                              </span>
                            </>
                          ) : (
                            <>
                              <span className="status status-inactive">
                                {" "}
                                Not Listed{" "}
                              </span>
                            </>
                          )}
                        </td>
                        <td>
                          {text(formatDate(item1?.CreatedAt, "02 Jan 2006"))}
                        </td>
                        <td className="product-actions">
                          <a
                            className="product-action edit-action"
                            href={"/storage?edit=" + text(item1?.ID)}
                          >
                            {" "}
                            Edit{" "}
                          </a>
                          {truth(item1?.Status === "active") ? (
                            <>
                              <form method="POST" action="/storage">
                                <input
                                  type="hidden"
                                  name="action"
                                  defaultValue="unlist"
                                />
                                <input
                                  type="hidden"
                                  name="product_id"
                                  defaultValue={text(item1?.ID)}
                                />
                                <button
                                  type="submit"
                                  className="product-action listing-action"
                                >
                                  {" "}
                                  Unlist{" "}
                                </button>
                              </form>
                            </>
                          ) : (
                            <>
                              <form method="POST" action="/storage">
                                <input
                                  type="hidden"
                                  name="action"
                                  defaultValue="relist"
                                />
                                <input
                                  type="hidden"
                                  name="product_id"
                                  defaultValue={text(item1?.ID)}
                                />
                                <button
                                  type="submit"
                                  className="product-action listing-action"
                                >
                                  {" "}
                                  List{" "}
                                </button>
                              </form>
                            </>
                          )}
                          <form
                            method="POST"
                            action="/storage"
                            className="delete-form"
                          >
                            <input
                              type="hidden"
                              name="action"
                              defaultValue="delete"
                            />
                            <input
                              type="hidden"
                              name="product_id"
                              defaultValue={text(item1?.ID)}
                            />
                            <button
                              type="submit"
                              className="product-action delete-action"
                            >
                              {" "}
                              Delete{" "}
                            </button>
                          </form>
                        </td>
                      </tr>
                    </Fragment>
                  ))
                ) : (
                  <>
                    <tr>
                      <td colSpan="10">
                        {" "}
                        No products in storage yet — add your first agricultural
                        product above.{" "}
                      </td>
                    </tr>
                  </>
                )}
              </tbody>
            </table>
          </div>
        </section>
        <div
          id="modal"
          className={"modal" + (editor.open ? " modal-open" : "")}
          onClick={(event) => {
            if (event.target === event.currentTarget) editor.close();
          }}
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-title"
        >
          <form
            id="storage-form"
            ref={editor.form}
            key={editProduct?.ID || "new"}
            onSubmit={editor.submit}
            method="POST"
            action="/storage"
            encType="multipart/form-data"
          >
            <h2 id="modal-title">
              {truth(editProduct) ? <> Edit Product </> : <> Add Product </>}
            </h2>
            <input
              type="hidden"
              name="action"
              id="form-action"
              defaultValue={truth(editProduct) ? "update" : "create"}
            />
            <input
              type="hidden"
              name="product_id"
              id="product-id"
              defaultValue={truth(editProduct) ? text(editProduct?.ID) : ""}
            />
            <label htmlFor="produce"> Product Name </label>
            <input
              type="text"
              name="produce"
              id="produce"
              defaultValue={truth(editProduct) ? text(editProduct?.Name) : ""}
              placeholder="e.g. Maize, Fish, Eggs, Honey"
              maxLength="150"
              required={true}
            />
            <label htmlFor="description"> Description </label>
            <Textarea
              name="description"
              id="description"
              placeholder="Describe your product..."
              maxLength="1000"
              rows="4"
            >
              {truth(editProduct) ? (
                <>{text(editProduct?.Description)}</>
              ) : null}
            </Textarea>
            <label htmlFor="category"> Agricultural Category </label>
            <Select name="category" id="category" required={true}>
              <option value=""> Select category </option>
              <option
                value="crops"
                selected={
                  truth(editProduct) && truth(editProduct?.Category === "crops")
                }
              >
                {" "}
                Crops{" "}
              </option>
              <option
                value="livestock"
                selected={
                  truth(editProduct) &&
                  truth(editProduct?.Category === "livestock")
                }
              >
                {" "}
                Livestock{" "}
              </option>
              <option
                value="poultry"
                selected={
                  truth(editProduct) &&
                  truth(editProduct?.Category === "poultry")
                }
              >
                {" "}
                Poultry{" "}
              </option>
              <option
                value="fisheries"
                selected={
                  truth(editProduct) &&
                  truth(editProduct?.Category === "fisheries")
                }
              >
                {" "}
                Fisheries / Aquaculture{" "}
              </option>
              <option
                value="beekeeping"
                selected={
                  truth(editProduct) &&
                  truth(editProduct?.Category === "beekeeping")
                }
              >
                {" "}
                Beekeeping{" "}
              </option>
              <option
                value="other"
                selected={
                  truth(editProduct) && truth(editProduct?.Category === "other")
                }
              >
                {" "}
                Other Agriculture{" "}
              </option>
            </Select>
            <label htmlFor="quantity"> How much do you have? </label>
            <input
              type="number"
              step="0.01"
              min="0.01"
              name="quantity"
              id="quantity"
              defaultValue={
                truth(editProduct) ? text(editProduct?.Quantity) : ""
              }
              required={true}
            />
            <label htmlFor="unit"> How is it sold? </label>
            <input
              type="text"
              name="unit"
              id="unit"
              defaultValue={truth(editProduct) ? text(editProduct?.Unit) : ""}
              placeholder="kg, bag, crate, litre..."
              maxLength="50"
              required={true}
            />
            <label htmlFor="location"> Farm / Production Location </label>
            <input
              type="text"
              name="location"
              id="location"
              defaultValue={
                truth(editProduct) ? text(editProduct?.Location) : ""
              }
              placeholder="e.g. Otukpo farm"
              maxLength="200"
              required={true}
            />
            <input
              type="hidden"
              name="community"
              id="community"
              defaultValue={editProduct?.CommunityID ?? ""}
            />
            <input
              type="hidden"
              name="latitude"
              id="latitude"
              defaultValue={
                truth(editProduct) ? text(editProduct?.Latitude) : ""
              }
            />
            <input
              type="hidden"
              name="longitude"
              id="longitude"
              defaultValue={
                truth(editProduct) ? text(editProduct?.Longitude) : ""
              }
            />
            <input
              type="hidden"
              name="location_accuracy"
              id="location-accuracy"
              defaultValue={
                truth(editProduct) ? text(editProduct?.LocationAccuracy) : ""
              }
            />
            <button
              type="button"
              id="capture-location"
              onClick={editor.capture}
              className="location-button"
            >
              {" "}
              Capture Farm Location{" "}
            </button>
            <small id="location-status" role="status">
              {editor.status || (
                <>
                  {truth(
                    truth(editProduct) &&
                      truth(
                        hasCoordinates(
                          editProduct?.Latitude,
                          editProduct?.Longitude,
                        ),
                      ),
                  ) ? (
                    <>
                      {" "}
                      Saved GPS location will be kept unless you capture it
                      again.{" "}
                    </>
                  ) : (
                    <>
                      {" "}
                      Capture GPS while physically at the farm or production
                      location.{" "}
                    </>
                  )}
                </>
              )}
            </small>
            <label htmlFor="price">
              {" "}
              Price for each kg, bag, crate, etc. (₦){" "}
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              name="price"
              id="price"
              defaultValue={truth(editProduct) ? text(editProduct?.Price) : ""}
              placeholder="Example: 5000"
              required={true}
            />
            <label htmlFor="produce-image"> Product Photo </label>
            <input
              type="file"
              name="produce_image"
              id="produce-image"
              accept="image/jpeg,image/png,image/webp"
              required={truth(!truth(editProduct))}
            />
            <small id="image-help">
              {truth(editProduct) ? (
                <> Leave empty to keep the current product image. </>
              ) : (
                <> A product photo is required. </>
              )}
            </small>
            <label className="checkbox-row">
              <input
                type="checkbox"
                name="list_for_sale"
                id="list-for-sale"
                defaultChecked={
                  truth(editProduct) && truth(editProduct?.Status === "active")
                }
              />
              <span> List this product on the Marketplace </span>
            </label>
            <div className="modal-actions">
              <button type="submit" className="btn btn-primary">
                {" "}
                Save Product{" "}
              </button>
              <a
                href="/storage"
                id="close-btn"
                className="cancel-action"
                onClick={(event) => {
                  event.preventDefault();
                  editor.close();
                }}
              >
                {" "}
                Cancel{" "}
              </a>
            </div>
          </form>
        </div>{" "}
      </main>
    </>
  );
}
