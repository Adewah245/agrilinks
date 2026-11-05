import { WorkspaceSidebar } from "../components/Workspace.jsx";
import { Select } from "../components/FormControls.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  money,
  humanBadge,
  formatDate,
} from "../lib/view.js";
import mountProfile from "../behaviors/profile.js";
import mountProfileLocations from "../behaviors/profile-locations.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function ProfilePage({ data }) {
  usePageBehaviors([mountProfile, mountProfileLocations], data);
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/profile">
        <a href="/dashboard"> Dashboard </a>
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <a href="/storage"> Storage </a>
          </>
        ) : null}
        <a href="/marketplace"> Marketplace </a>
        <a href="/negotiations"> Negotiations </a>
        <a href="/chat"> Chat </a>
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <a href="/ai-assistant"> AI Assistant </a>
          </>
        ) : null}
        <a href="/cart"> Cart </a>
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
        <a href="/profile" aria-current="page">
          {" "}
          Profile{" "}
        </a>
        <a href="/logout"> Logout </a>
        <a href="/wallet">Wallet</a>
      </WorkspaceSidebar>
      <main className="profile">
        <section className="profile-header" aria-labelledby="profile-name">
          {truth(data?.PhotoURL) ? (
            <>
              <img
                src={text(data?.PhotoURL)}
                alt={text(data?.FullName) + " profile photo"}
                className="profile-photo"
              />
            </>
          ) : (
            <>
              <div className="profile-photo-placeholder" aria-hidden="true">
                {" "}
                👤{" "}
              </div>
            </>
          )}
          <div className="profile-header-content">
            <h1 id="profile-name">{text(data?.FullName)}</h1>
            <p className="profile-role">
              {truth(data?.IsBuyer) ? (
                <> Buyer </>
              ) : truth(data?.IsOrganisation) ? (
                <> Organisation </>
              ) : truth(data?.IsMinistry) ? (
                <> Ministry </>
              ) : (
                <> Farmer </>
              )}
              {truth(data?.Location) ? <> • {text(data?.Location)}</> : null}
            </p>
            {truth(data?.Badges) ? (
              <>
                <p className="profile-badges" aria-label="Trust badges">
                  {truth(data?.Badges)
                    ? items(data?.Badges).map((item1, index1) => (
                        <Fragment key={index1}>
                          <span className="profile-badge">
                            {text(humanBadge(item1?.BadgeType))}
                          </span>
                        </Fragment>
                      ))
                    : null}
                </p>
              </>
            ) : null}
            {truth(data?.IsOwner) ? (
              <>
                {truth(data?.IsAdministrator) ? (
                  <>
                    <p>
                      <a href="/trust">Manage trust badges</a>
                    </p>
                  </>
                ) : null}
                <button
                  type="button"
                  id="edit-profile-btn"
                  className="profile-edit-button"
                >
                  {" "}
                  Edit Profile{" "}
                </button>
              </>
            ) : null}
          </div>
        </section>
        <section className="profile-grid" aria-label="Profile information">
          <div className="card">
            <h2> Personal Information </h2>
            <p>
              <strong>Full Name:</strong>
              {text(data?.FullName)}
            </p>
            <p>
              <strong>Phone:</strong>
              {text(data?.Phone)}
            </p>
            {truth(data?.Email) ? (
              <>
                <p>
                  <strong>Email:</strong>
                  {text(data?.Email)}
                </p>
              </>
            ) : null}
            {truth(data?.Location) ? (
              <>
                <p>
                  <strong>Location:</strong>
                  {text(data?.Location)}
                </p>
              </>
            ) : null}
            <p>
              <strong>Account Type:</strong>
              {truth(data?.IsBuyer) ? (
                <> Buyer </>
              ) : truth(data?.IsOrganisation) ? (
                <> Organisation </>
              ) : truth(data?.IsMinistry) ? (
                <> Ministry </>
              ) : (
                <> Farmer </>
              )}
            </p>
            <p>
              <strong>Member Since:</strong>
              {text(formatDate(data?.CreatedAt, "January 2006"))}
            </p>
          </div>
          {truth(data?.FarmingTypes) ? (
            <>
              <div className="card">
                <h2> Farming Activities </h2>
                <div className="farming-types">
                  {truth(data?.FarmingTypes)
                    ? items(data?.FarmingTypes).map((item2, index2) => (
                        <Fragment key={index2}>
                          <span className="farming-type">{text(item2)}</span>
                        </Fragment>
                      ))
                    : null}
                </div>
              </div>
            </>
          ) : null}
          {truth(data?.IsOwner) ? (
            <>
              {truth(data?.IsBuyer) ? (
                <>
                  <div className="card">
                    <h2> Purchase History </h2>
                    {truth(data?.Purchases) ? (
                      <>
                        <div className="table-container">
                          <div
                            className="table-scroll"
                            role="region"
                            aria-label="Profile table"
                            tabIndex="0"
                          >
                            <table>
                              <thead>
                                <tr>
                                  <th> Product </th>
                                  <th> Seller </th>
                                  <th> How much </th>
                                  <th> Total </th>
                                  <th> Date </th>
                                </tr>
                              </thead>
                              <tbody>
                                {truth(data?.Purchases)
                                  ? items(data?.Purchases).map(
                                      (item3, index3) => (
                                        <Fragment key={index3}>
                                          <tr>
                                            <td>{text(item3?.ProductName)}</td>
                                            <td>{text(item3?.SellerName)}</td>
                                            <td>
                                              {text(item3?.Quantity)}
                                              {text(item3?.ProductUnit)}
                                            </td>
                                            <td>
                                              {" "}
                                              ₦{text(money(item3?.TotalPrice))}
                                            </td>
                                            <td>
                                              {text(
                                                formatDate(
                                                  item3?.CreatedAt,
                                                  "02 Jan 2006",
                                                ),
                                              )}
                                            </td>
                                          </tr>
                                        </Fragment>
                                      ),
                                    )
                                  : null}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      </>
                    ) : (
                      <>
                        <p>
                          {" "}
                          No purchases yet.{" "}
                          <a href="/marketplace"> Browse the marketplace </a>
                        </p>
                      </>
                    )}
                  </div>
                </>
              ) : (
                <>
                  <div className="card">
                    <h2> My Products </h2>
                    {truth(data?.Products) ? (
                      <>
                        <div className="table-container">
                          <div
                            className="table-scroll"
                            role="region"
                            aria-label="Profile table"
                            tabIndex="0"
                          >
                            <table>
                              <thead>
                                <tr>
                                  <th> Product </th>
                                  <th> Category </th>
                                  <th> How much </th>
                                  <th> Status </th>
                                  <th> Price </th>
                                </tr>
                              </thead>
                              <tbody>
                                {truth(data?.Products)
                                  ? items(data?.Products).map(
                                      (item4, index4) => (
                                        <Fragment key={index4}>
                                          <tr>
                                            <td>{text(item4?.Name)}</td>
                                            <td>{text(item4?.Category)}</td>
                                            <td>
                                              {text(item4?.Quantity)}
                                              {text(item4?.Unit)}
                                            </td>
                                            <td>
                                              {truth(
                                                item4?.Status === "active",
                                              ) ? (
                                                <> Available </>
                                              ) : truth(
                                                  item4?.Status === "sold_out",
                                                ) ? (
                                                <> Sold Out </>
                                              ) : (
                                                <> Inactive </>
                                              )}
                                            </td>
                                            <td>
                                              {" "}
                                              ₦{text(money(item4?.Price))} for
                                              each {text(item4?.Unit)}
                                            </td>
                                          </tr>
                                        </Fragment>
                                      ),
                                    )
                                  : null}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      </>
                    ) : (
                      <>
                        <p>
                          {" "}
                          No product added yet.{" "}
                          <a href="/storage"> Add your first product </a>
                        </p>
                      </>
                    )}
                  </div>
                  <div className="card">
                    <h2> Recent Sales </h2>
                    {truth(data?.Sales) ? (
                      <>
                        <div className="table-container">
                          <div
                            className="table-scroll"
                            role="region"
                            aria-label="Profile table"
                            tabIndex="0"
                          >
                            <table>
                              <thead>
                                <tr>
                                  <th> Product </th>
                                  <th> Buyer </th>
                                  <th> How much </th>
                                  <th> Total </th>
                                  <th> Date </th>
                                </tr>
                              </thead>
                              <tbody>
                                {truth(data?.Sales)
                                  ? items(data?.Sales).map((item5, index5) => (
                                      <Fragment key={index5}>
                                        <tr>
                                          <td>{text(item5?.ProductName)}</td>
                                          <td>{text(item5?.BuyerName)}</td>
                                          <td>
                                            {text(item5?.Quantity)}
                                            {text(item5?.ProductUnit)}
                                          </td>
                                          <td>
                                            {" "}
                                            ₦{text(money(item5?.TotalPrice))}
                                          </td>
                                          <td>
                                            {text(
                                              formatDate(
                                                item5?.CreatedAt,
                                                "02 Jan 2006",
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
                        </div>
                      </>
                    ) : (
                      <>
                        <p> No sales yet. </p>
                      </>
                    )}
                  </div>
                </>
              )}
            </>
          ) : null}
          {truth(data?.IsOwner) ? (
            <>
              <div className="card">
                <h2> Security </h2>
                <p>
                  {" "}
                  Keep your account secure by changing your password
                  regularly.{" "}
                </p>
                <a href="/forgot-password" className="profile-security-button">
                  {" "}
                  Change Password{" "}
                </a>
              </div>
            </>
          ) : null}
        </section>
      </main>
      {truth(data?.IsOwner) ? (
        <>
          <div
            id="edit-modal"
            className="modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="edit-profile-title"
          >
            <form
              method="POST"
              action="/profile/edit"
              encType="multipart/form-data"
            >
              <h2 id="edit-profile-title"> Edit Profile </h2>
              <label htmlFor="profile-full-name"> Full Name </label>
              <input
                id="profile-full-name"
                type="text"
                name="full_name"
                defaultValue={text(data?.FullName)}
                autoComplete="name"
                required={true}
              />
              <label htmlFor="profile-phone"> Phone </label>
              <input
                id="profile-phone"
                type="tel"
                name="phone"
                defaultValue={text(data?.Phone)}
                autoComplete="tel"
                required={true}
              />
              <label htmlFor="profile-email"> Email </label>
              <input
                id="profile-email"
                type="email"
                name="email"
                defaultValue={text(data?.Email)}
                autoComplete="email"
              />
              <label htmlFor="profile-state">State</label>
              <Select
                id="profile-state"
                name="state"
                data-selected={text(data?.State)}
                required={true}
              >
                <option value={text(data?.State)}>{text(data?.State)}</option>
              </Select>
              <label htmlFor="profile-lga">LGA</label>
              <Select
                id="profile-lga"
                name="lga"
                data-selected={text(data?.LGA)}
                required={true}
              >
                <option value={text(data?.LGA)}>{text(data?.LGA)}</option>
              </Select>
              <label htmlFor="profile-location">Community or town</label>
              <input
                id="profile-location"
                name="location"
                defaultValue={text(data?.Location)}
                maxLength="200"
                required={true}
              />
              <label htmlFor="profile-password">
                Current password (needed when changing your phone or email)
              </label>
              <input
                id="profile-password"
                name="current_password"
                type="password"
                autoComplete="current-password"
              />
              <p id="profile-location-error" role="alert"></p>
              <label htmlFor="profile-passport"> Profile Photograph </label>
              <input
                id="profile-passport"
                type="file"
                name="passport"
                accept="image/jpeg,image/png,image/webp"
              />
              <small> Leave empty to keep your current photograph. </small>
              <div className="modal-actions">
                <button type="submit"> Save </button>
                <button type="button" id="cancel-edit-btn">
                  {" "}
                  Cancel{" "}
                </button>
              </div>
            </form>
          </div>
        </>
      ) : null}
      {truth(data?.IsOwner) ? <></> : null}
    </div>
  );
}
