import { WorkspaceSidebar } from "../components/Workspace.jsx";
import FarmingLearningHub from "../components/FarmingLearningHub.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  withValue,
  text,
  money,
  money0,
  humanOrderStatus,
  formatDate,
  printf,
} from "../lib/view.js";
import mountDashboard from "../behaviors/dashboard.js";
import mountWeather from "../behaviors/weather.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function DashboardPage({ data }) {
  usePageBehaviors([mountDashboard, mountWeather], data);
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/dashboard">
        {truth(data?.IsStaff) ? (
          <>
            <a href="/admin">Admin workspace</a>
          </>
        ) : null}
        <a className="active" href="/dashboard" aria-current="page">
          {" "}
          Dashboard{" "}
        </a>
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <a href="/storage"> Storage </a>
          </>
        ) : null}
        <a href="/marketplace"> Marketplace </a>
        {truth(data?.IsBuyer) ? (
          <>
            <a href="/payments"> Payments </a>
            <a href="/deliveries"> Deliveries </a>
          </>
        ) : null}
        <a href="/chat"> Chat </a>
        <a href="/learning"> Learn with AI </a>
        {truth(data?.IsBuyer) ? (
          <>
            <a href="/cart" className="cart-nav-link">
              {" "}
              Cart{" "}
              {truth(data?.CartItemCount) ? (
                <>
                  <span className="cart-badge">
                    {text(data?.CartItemCount)}
                  </span>
                </>
              ) : null}
            </a>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <a href="/ai-assistant"> AI Assistant </a>
            <a href="/sales"> Sales </a>
          </>
        ) : null}
        <a href="/profile"> Profile </a>
        <a
          href="/notifications"
          className="notification-nav-link"
          aria-label="Notifications"
        >
          Notifications
          {truth(data?.UnreadNotifications) ? (
            <>
              <span className="notification-nav-badge">
                {text(data?.UnreadNotifications)}
              </span>
            </>
          ) : null}
        </a>
        <a href="/logout"> Log out </a>
        <a href="/wallet">Wallet</a>
      </WorkspaceSidebar>
      <main className="dashboard">
        <section className="dashboard-header">
          <div className="dashboard-intro">
            <div className="dashboard-datetime">
              <p className="eyebrow" id="today">
                {" "}
                Today{" "}
              </p>
              <p
                className="current-time"
                id="currentTime"
                aria-live="polite"
              ></p>
            </div>
            <h1>
              <span id="greeting"> Welcome back </span>, {text(data?.FullName)}
            </h1>
            {truth(data?.IsBuyer) ? (
              <>
                <p className="header-copy">
                  {" "}
                  Find agricultural products from trusted farmers and sellers
                  and keep track of your purchases.{" "}
                </p>
              </>
            ) : (
              <>
                <p className="header-copy">
                  {" "}
                  Keep an eye on your farm, sell your products, and find help
                  with your next steps.{" "}
                </p>
              </>
            )}
          </div>
          <div className="header-actions">
            {truth(data?.IsBuyer) ? (
              <>
                <a className="primary-action" href="/marketplace">
                  {" "}
                  Browse marketplace{" "}
                </a>
              </>
            ) : (
              <>
                <a className="primary-action" href="/storage">
                  {" "}
                  Add product{" "}
                </a>
              </>
            )}
          </div>
        </section>
        <section className="metrics" aria-label="Account overview">
          {truth(data?.IsBuyer) ? (
            <>
              <a href="/profile" className="metric-card">
                <span className="metric-label"> Purchases made </span>
                <strong
                  className="counter"
                  data-target={text(data?.PurchasesMade)}
                >
                  {" "}
                  0{" "}
                </strong>
                <span className="metric-detail"> Completed orders </span>
              </a>
              <a href="/profile" className="metric-card">
                <span className="metric-label"> Total spent </span>
                <strong> ₦{text(money(data?.TotalSpent))}</strong>
                <span className="metric-detail"> Across all purchases </span>
              </a>
              <a href="/marketplace" className="metric-card">
                <span className="metric-label"> Available listings </span>
                <strong>{text(data?.AvailableProduce)}</strong>
                <span className="metric-detail">
                  {" "}
                  Agricultural products currently available{" "}
                </span>
              </a>
              <div className="metric-card">
                <span className="metric-label"> Account status </span>
                <strong className="status-positive"> Verified </strong>
                <span className="metric-detail">
                  {" "}
                  Trusted Agrilinks Nigeria buyer{" "}
                </span>
              </div>
            </>
          ) : (
            <>
              <a href="/storage" className="metric-card">
                <span className="metric-label"> Products in storage </span>
                <strong
                  className="counter"
                  data-target={text(data?.ProduceInStorage)}
                >
                  {" "}
                  0{" "}
                </strong>
                <span className="metric-detail"> Product items recorded </span>
              </a>
              <a href="/storage" className="metric-card">
                <span className="metric-label"> Active listings </span>
                <strong
                  className="counter"
                  data-target={text(data?.ListingsActive)}
                >
                  {" "}
                  0{" "}
                </strong>
                <span className="metric-detail">
                  {" "}
                  Visible to buyers · your account{" "}
                </span>
              </a>
              <a href="/ai-diagnosis-history" className="metric-card">
                <span className="metric-label"> AI health checks </span>
                <strong
                  className="counter"
                  data-target={text(data?.AIDiagnosesThisMonth)}
                >
                  {" "}
                  0{" "}
                </strong>
                <span className="metric-detail">
                  {" "}
                  Completed this month · your account{" "}
                </span>
              </a>
              <a href="/profile" className="metric-card">
                <span className="metric-label"> Sales revenue · 30 days </span>
                <strong> ₦{text(money(data?.Revenue30))}</strong>
                <span className="metric-detail">
                  {text(data?.CompletedSales)} completed sales · platform
                  records{" "}
                </span>
              </a>
            </>
          )}
        </section>
        <section
          className="quick-actions"
          aria-labelledby="quick-actions-title"
        >
          <div className="section-heading">
            <div>
              <p className="eyebrow"> Shortcuts </p>
              <h2 id="quick-actions-title"> Continue working </h2>
            </div>
          </div>
          <div className="action-grid">
            <FarmingLearningHub className="action-card">
              <span>Learning</span>
              <h3>Learn farming</h3>
              <p>Explore farming topics and ask the AI learning guide.</p>
            </FarmingLearningHub>
            {truth(data?.IsBuyer) ? (
              <>
                <a href="/marketplace" className="action-card">
                  <span> Market </span>
                  <h3> Explore agricultural products </h3>
                  <p>
                    {" "}
                    Browse available products from farmers and agricultural
                    sellers.{" "}
                  </p>
                </a>
                <a href="/profile" className="action-card">
                  <span> Orders </span>
                  <h3> View purchases </h3>
                  <p> Review your orders and transactions. </p>
                </a>
                <a href="/profile" className="action-card">
                  <span> Account </span>
                  <h3> Manage profile </h3>
                  <p> Update your account and preferences. </p>
                </a>
              </>
            ) : (
              <>
                <a href="/storage" className="action-card">
                  <span> Products </span>
                  <h3> Manage products </h3>
                  <p> Add products and keep availability up to date. </p>
                </a>
                <a href="/marketplace" className="action-card">
                  <span> Market </span>
                  <h3> List products </h3>
                  <p> Put your agricultural products in front of buyers. </p>
                </a>
                <a href="/ai-assistant" className="action-card">
                  <span> Farm health </span>
                  <h3> Ask the AI assistant </h3>
                  <p>
                    {" "}
                    Analyse a farm problem, image, or get agricultural
                    advice.{" "}
                  </p>
                </a>
                <a href="/marketplace" className="action-card">
                  <span> Sales </span>
                  <h3> Manage negotiations </h3>
                  <p> Review offers and negotiate with buyers. </p>
                </a>
              </>
            )}
          </div>
        </section>
        <section
          className="panel marketplace-slider"
          aria-labelledby="marketplace-slider-title"
        >
          <div className="section-heading">
            <div>
              <p className="eyebrow">Available now</p>
              <h2 id="marketplace-slider-title">Products in the marketplace</h2>
            </div>
            <div
              className="slider-controls"
              aria-label="Product slider controls"
            >
              <button
                type="button"
                data-product-slide="previous"
                aria-label="Show previous products"
              >
                ‹
              </button>
              <button
                type="button"
                data-product-slide="next"
                aria-label="Show more products"
              >
                ›
              </button>
            </div>
          </div>
          <div
            className="marketplace-product-track"
            id="marketplace-product-track"
          >
            {truth(data?.MarketplaceProducts) ? (
              items(data?.MarketplaceProducts).map((item1, index1) => (
                <Fragment key={index1}>
                  <article className="marketplace-product-card">
                    <a
                      className="marketplace-product-image"
                      href={"/product?id=" + text(item1?.ID)}
                      aria-label={"View " + text(item1?.Name)}
                    >
                      {truth(item1?.PrimaryImageURL) ? (
                        <>
                          <img
                            src={text(item1?.PrimaryImageURL)}
                            alt={text(item1?.Name)}
                            loading="lazy"
                          />
                        </>
                      ) : (
                        <>
                          <span aria-hidden="true">🌾</span>
                        </>
                      )}
                    </a>
                    <div className="marketplace-product-details">
                      <p className="marketplace-product-category">
                        {text(item1?.Category)}
                      </p>
                      <h3>
                        <a href={"/product?id=" + text(item1?.ID)}>
                          {text(item1?.Name)}
                        </a>
                      </h3>
                      <p className="marketplace-product-seller">
                        {text(item1?.SellerName)}
                      </p>
                      <strong>
                        ₦{text(money(item1?.Price))}
                        <span>/ {text(item1?.Unit)}</span>
                      </strong>
                      <p className="marketplace-product-quantity">
                        {text(printf("%.0f", item1?.Quantity))}
                        {text(item1?.Unit)} available
                      </p>
                      <a
                        className="marketplace-product-link"
                        href={"/product?id=" + text(item1?.ID)}
                      >
                        View product
                      </a>
                    </div>
                  </article>
                </Fragment>
              ))
            ) : (
              <>
                <div className="empty-state marketplace-empty">
                  <p>No products are available in the marketplace right now.</p>
                  <a href="/marketplace">Visit the marketplace</a>
                </div>
              </>
            )}
          </div>
        </section>
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section
              className="panel storage-overview"
              aria-labelledby="storage-overview-title"
            >
              <div className="section-heading">
                <div>
                  <p className="eyebrow">Your stock</p>
                  <h2 id="storage-overview-title">Products in storage</h2>
                </div>
                <a href="/storage">Manage products</a>
              </div>
              <div className="produce-list">
                {truth(data?.StorageCrops) ? (
                  items(data?.StorageCrops).map((item2, index2) => (
                    <Fragment key={index2}>
                      <div className="produce-row">
                        <div className="produce-info">
                          <span className="produce-icon" aria-hidden="true">
                            {truth(item2?.Status === "active") ? (
                              <>🌾</>
                            ) : (
                              <>📦</>
                            )}
                          </span>
                          <div>
                            <h3>{text(item2?.Name)}</h3>
                            <span>
                              {truth(item2?.Status === "active") ? (
                                <>Listed for sale</>
                              ) : (
                                <>Not listed</>
                              )}
                            </span>
                          </div>
                        </div>
                        <div className="produce-quantity">
                          <strong>
                            {text(printf("%.0f", item2?.Quantity))}
                          </strong>
                          <span>{text(item2?.Unit)}</span>
                        </div>
                        <span className="stock-status healthy">
                          {truth(item2?.Quantity === 0.0) ? (
                            <>Empty</>
                          ) : truth(item2?.Status === "active") ? (
                            <>Listed</>
                          ) : (
                            <>Stored</>
                          )}
                        </span>
                      </div>
                    </Fragment>
                  ))
                ) : (
                  <>
                    <div className="empty-state">
                      <p>No products in storage yet.</p>
                      <a href="/storage">Add your first product</a>
                    </div>
                  </>
                )}
              </div>
            </section>
          </>
        ) : null}
        <section
          className={
            "panel " +
            (truth(data?.IsBuyer) ? "dashboard-weather" : "weather-card") +
            " device-weather"
          }
          data-device-weather=""
          aria-labelledby="device-weather-title"
          data-weather-state="waiting"
        >
          <div className="section-heading">
            <div>
              <p className="eyebrow">Where you are</p>
              <h2 id="device-weather-title">Today's weather</h2>
              <span className="weather-source" data-weather-location="">
                Your device location
              </span>
              <span
                className="weather-updated"
                data-weather-place-status=""
                hidden={true}
                role="status"
              ></span>
              <span
                className="weather-updated"
                data-weather-location-credit=""
                hidden={true}
              >
                Place names by{" "}
                <a
                  href="https://www.bigdatacloud.com/free-api/free-reverse-geocode-to-city-api"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  BigDataCloud
                </a>
              </span>
              <span
                className="weather-updated"
                data-weather-updated=""
                hidden={true}
              ></span>
            </div>
          </div>
          <p
            className="weather-location-status"
            data-weather-status=""
            role="status"
            aria-live="polite"
          >
            Allow location access when your browser asks to see weather where
            you are.
          </p>
          <div data-weather-values="" hidden={true}>
            <div className="weather-main">
              <span
                className="weather-icon"
                data-weather-icon=""
                aria-hidden="true"
              ></span>
              <div>
                <strong
                  className="temperature"
                  data-weather-temperature=""
                ></strong>
                <span data-weather-summary=""></span>
              </div>
            </div>
            <div className="weather-details">
              <div>
                <span>Humidity</span>
                <strong data-weather-humidity=""></strong>
              </div>
              <div>
                <span>Rain chance this hour</span>
                <strong data-weather-rain=""></strong>
              </div>
              <div>
                <span>Conditions</span>
                <strong data-weather-conditions=""></strong>
              </div>
            </div>
            <div className="weather-recommendation">
              <strong>Farm recommendation</strong>
              <p data-weather-recommendation=""></p>
            </div>
            <p
              className="weather-location-accuracy"
              data-weather-accuracy=""
            ></p>
          </div>
          <button
            type="button"
            className="weather-location-button"
            data-weather-refresh=""
          >
            Use my location
          </button>
          <noscript>
            <p>
              Enable JavaScript and allow location access to see your local
              weather.
            </p>
          </noscript>
        </section>
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section
              className="panel dashboard-market-prices"
              aria-labelledby="dashboard-market-prices-title"
            >
              <div className="section-heading">
                <div>
                  <p className="eyebrow">Local market</p>
                  <h2 id="dashboard-market-prices-title">
                    Market price and margin
                  </h2>
                  <p className="section-description">
                    Compare your average listing price with the local median.
                    This is a price difference, not profit after costs.
                  </p>
                </div>
                <a href="/market-insights">More insights</a>
              </div>
              <div className="dashboard-price-grid">
                {truth(data?.ProduceDemandPreview) ? (
                  items(data?.ProduceDemandPreview).map((item3, index3) => (
                    <Fragment key={index3}>
                      {truth(item3?.PriceAvailable) ? (
                        <>
                          <article className="dashboard-price-card">
                            <h3>{text(item3?.ProduceType)}</h3>
                            <dl>
                              <div>
                                <dt>Local median</dt>
                                <dd>
                                  ₦{text(money0(item3?.LocalPriceMedian))}
                                </dd>
                              </div>
                              <div>
                                <dt>Your average price</dt>
                                <dd>₦{text(money0(item3?.FarmerPrice))}</dd>
                              </div>
                              <div>
                                <dt>Difference</dt>
                                <dd>
                                  {text(
                                    printf(
                                      "%+.0f%%",
                                      item3?.PriceDifferencePercent,
                                    ),
                                  )}
                                </dd>
                              </div>
                            </dl>
                            <p>{text(item3?.PricePosition)}</p>
                          </article>
                        </>
                      ) : null}
                    </Fragment>
                  ))
                ) : (
                  <>
                    <div className="empty-state">
                      <p>Local price comparisons are not available yet.</p>
                      <a href="/market-insights">Open market insights</a>
                    </div>
                  </>
                )}
              </div>
            </section>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section
              className="panel farming-overview"
              aria-labelledby="farming-overview-title"
            >
              <div className="section-heading">
                <div>
                  <p className="eyebrow"> Farming Activities </p>
                  <h2 id="farming-overview-title"> Your Farm Types </h2>
                </div>
                <a href="/profile"> Manage profile </a>
              </div>
              <div className="farm-type-grid">
                {truth(data?.FarmingTypes) ? (
                  items(data?.FarmingTypes).map((item4, index4) => (
                    <Fragment key={index4}>
                      <article className="farm-type-card">
                        <div className="farm-type-icon" aria-hidden="true">
                          {truth(item4 === "Crop Farming") ? (
                            <> 🌾 </>
                          ) : truth(item4 === "Animal Farming") ? (
                            <> 🐄 </>
                          ) : truth(item4 === "Poultry Farming") ? (
                            <> 🐔 </>
                          ) : truth(item4 === "Fish Farming") ? (
                            <> 🐟 </>
                          ) : truth(item4 === "Apiculture") ? (
                            <> 🐝 </>
                          ) : truth(item4 === "Pig Farming") ? (
                            <> 🐖 </>
                          ) : truth(item4 === "Goat Farming") ? (
                            <> 🐐 </>
                          ) : truth(item4 === "Sheep Farming") ? (
                            <> 🐑 </>
                          ) : (
                            <> 🌱 </>
                          )}
                        </div>
                        <div className="farm-type-content">
                          <h3>{text(item4)}</h3>
                          <p> Manage your {text(item4)} activities. </p>
                        </div>
                      </article>
                    </Fragment>
                  ))
                ) : (
                  <>
                    <article className="farm-type-card farm-type-empty">
                      <div className="farm-type-icon" aria-hidden="true">
                        {" "}
                        🌱{" "}
                      </div>
                      <div className="farm-type-content">
                        <h3> Farming type not selected </h3>
                        <p>
                          {" "}
                          Select your farming activities from your profile.{" "}
                        </p>
                      </div>
                      <a href="/profile"> Update profile </a>
                    </article>
                  </>
                )}
              </div>
            </section>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section
              className={
                "attention-banner priority-" + text(data?.PrimaryPriority?.Tone)
              }
              aria-label="Priority alert"
            >
              <div className="attention-icon" aria-hidden="true">
                {truth(data?.PrimaryPriority?.Tone === "danger") ? (
                  <> ! </>
                ) : truth(data?.PrimaryPriority?.Tone === "warning") ? (
                  <> ! </>
                ) : (
                  <> ✓ </>
                )}
              </div>
              <div className="attention-content">
                <span className="attention-label">
                  {text(data?.PrimaryPriority?.Label)}
                </span>
                <h2>{text(data?.PrimaryPriority?.Title)}</h2>
                <p>{text(data?.PrimaryPriority?.Detail)}</p>
              </div>
              <a
                href={text(data?.PrimaryPriority?.Href)}
                className="primary-action priority-action"
              >
                {text(data?.PrimaryPriority?.Action)}
              </a>
            </section>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section className="dashboard-grid dashboard-grid-single">
              <section
                className="panel farm-health"
                aria-labelledby="farm-readiness-title"
              >
                <div className="section-heading">
                  <div>
                    <p className="eyebrow"> Overview </p>
                    <h2 id="farm-readiness-title"> Operational readiness </h2>
                  </div>
                  <span className="health-status">
                    {text(data?.HealthStatus)}
                  </span>
                </div>
                <div className="health-score">
                  <div className="health-score-number">
                    {text(data?.HealthScore)}
                    <span> /100 </span>
                  </div>
                  <div className="health-score-copy">
                    <strong> Your farm readiness overview </strong>
                    <p>
                      {" "}
                      This is an operational indicator based on storage,
                      listings, AI checks, and completed sales—not a diagnosis
                      of crops, livestock, poultry, fish, or other farm
                      activities.{" "}
                    </p>
                  </div>
                </div>
                <div className="health-bars">
                  <div className="health-item">
                    <div className="health-item-header">
                      <span> Storage </span>
                      <strong>
                        {withValue(data?.HealthMetrics?.[0], (item5) => (
                          <>{text(item5?.Percent)}</>
                        ))}
                        %{" "}
                      </strong>
                    </div>
                    <div className="progress">
                      <span
                        className="progress-value"
                        data-width={
                          withValue(
                            data?.HealthMetrics?.[0],
                            (value6) => text(value6?.Percent),
                            () => "",
                          ) + "%"
                        }
                      ></span>
                    </div>
                  </div>
                  <div className="health-item">
                    <div className="health-item-header">
                      <span> Listings </span>
                      <strong>
                        {withValue(data?.HealthMetrics?.[1], (item7) => (
                          <>{text(item7?.Percent)}</>
                        ))}
                        %{" "}
                      </strong>
                    </div>
                    <div className="progress">
                      <span
                        className="progress-value"
                        data-width={
                          withValue(
                            data?.HealthMetrics?.[1],
                            (value8) => text(value8?.Percent),
                            () => "",
                          ) + "%"
                        }
                      ></span>
                    </div>
                  </div>
                  <div className="health-item">
                    <div className="health-item-header">
                      <span> Sales activity </span>
                      <strong>
                        {withValue(data?.HealthMetrics?.[3], (item9) => (
                          <>{text(item9?.Percent)}</>
                        ))}
                        %{" "}
                      </strong>
                    </div>
                    <div className="progress">
                      <span
                        className="progress-value"
                        data-width={
                          withValue(
                            data?.HealthMetrics?.[3],
                            (value10) => text(value10?.Percent),
                            () => "",
                          ) + "%"
                        }
                      ></span>
                    </div>
                  </div>
                </div>
              </section>
            </section>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section
              className="panel farming-details"
              aria-labelledby="farming-management-title"
            >
              <div className="section-heading">
                <div>
                  <p className="eyebrow"> Farm Management </p>
                  <h2 id="farming-management-title">
                    {" "}
                    Manage your farming activities{" "}
                  </h2>
                  <p className="section-description">
                    {" "}
                    Access tools based on the farming activities you selected
                    during registration.{" "}
                  </p>
                </div>
              </div>
              <div className="farming-detail-grid">
                {truth(data?.FarmingTypes)
                  ? items(data?.FarmingTypes).map((item11, index11) => (
                      <Fragment key={index11}>
                        {truth(item11 === "Crop Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🌾{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Crop Farming{" "}
                                </span>
                                <h3> Manage crops </h3>
                                <p>
                                  {" "}
                                  Manage crops, storage, product listings and
                                  crop health.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Animal Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐄{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Animal Farming{" "}
                                </span>
                                <h3> Manage animals </h3>
                                <p>
                                  {" "}
                                  Track livestock, health, feeding, breeding and
                                  sales.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Poultry Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐔{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Poultry Farming{" "}
                                </span>
                                <h3> Manage poultry </h3>
                                <p>
                                  {" "}
                                  Track birds, eggs, feed, health and poultry
                                  sales.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Fish Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐟{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Fish Farming{" "}
                                </span>
                                <h3> Manage fish farm </h3>
                                <p>
                                  {" "}
                                  Manage ponds, fish stock, feeding, water
                                  quality and harvest.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Apiculture") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐝{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Apiculture{" "}
                                </span>
                                <h3> Manage beekeeping </h3>
                                <p>
                                  {" "}
                                  Track colonies, honey production, hive health
                                  and sales.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Pig Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐖{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Pig Farming{" "}
                                </span>
                                <h3> Manage pigs </h3>
                                <p>
                                  {" "}
                                  Track pigs, feeding, health, breeding and
                                  sales.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Goat Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐐{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Goat Farming{" "}
                                </span>
                                <h3> Manage goats </h3>
                                <p>
                                  {" "}
                                  Track goats, feeding, health, breeding and
                                  sales.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : truth(item11 === "Sheep Farming") ? (
                          <>
                            <a href="/storage" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🐑{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {" "}
                                  Sheep Farming{" "}
                                </span>
                                <h3> Manage sheep </h3>
                                <p>
                                  {" "}
                                  Track sheep, feeding, health, breeding and
                                  sales.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        ) : (
                          <>
                            <a href="/profile" className="farming-detail-card">
                              <span
                                className="farming-detail-icon"
                                aria-hidden="true"
                              >
                                {" "}
                                🌱{" "}
                              </span>
                              <div>
                                <span className="farming-detail-label">
                                  {text(item11)}
                                </span>
                                <h3> Manage farming activity </h3>
                                <p>
                                  {" "}
                                  Manage your {text(item11)} activities on
                                  Agrilinks Nigeria.{" "}
                                </p>
                              </div>
                            </a>
                          </>
                        )}
                      </Fragment>
                    ))
                  : null}
              </div>
            </section>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section className="dashboard-grid">
              <section
                className="panel ai-insight"
                aria-labelledby="produce-health-title"
              >
                <div className="section-heading">
                  <div>
                    <p className="eyebrow"> AI insight </p>
                    <h2 id="produce-health-title"> Farm health </h2>
                  </div>
                  <a href="/ai-assistant"> Open AI </a>
                </div>
                <div className="ai-diagnosis">
                  {withValue(
                    data?.LatestDiagnosis,
                    (item12) => (
                      <>
                        <div className="crop-icon" aria-hidden="true">
                          {" "}
                          AI{" "}
                        </div>
                        <div className="diagnosis-content">
                          <span className="diagnosis-label">
                            {" "}
                            Latest diagnosis summary ·{" "}
                            {text(formatDate(item12?.CreatedAt, "2 Jan 2006"))}
                          </span>
                          <h3>
                            {truth(item12?.CategoryText) ? (
                              <>{text(item12?.CategoryText)}</>
                            ) : (
                              <> Farm health check </>
                            )}
                          </h3>
                          <p>{text(item12?.ResultText)}</p>
                          <a
                            className="diagnosis-full-link"
                            href="/ai-diagnosis-history"
                          >
                            {" "}
                            View full diagnosis →{" "}
                          </a>
                        </div>
                      </>
                    ),
                    () => (
                      <>
                        <div className="crop-icon" aria-hidden="true">
                          {" "}
                          AI{" "}
                        </div>
                        <div className="diagnosis-content">
                          <span className="diagnosis-label">
                            {" "}
                            No diagnosis yet{" "}
                          </span>
                          <h3> Run your first farm health check </h3>
                          <p>
                            {" "}
                            Describe a farming problem or upload media to
                            receive an AI-assisted analysis.{" "}
                          </p>
                        </div>
                      </>
                    ),
                  )}
                </div>
                <a href="/ai-diagnosis-history" className="panel-action">
                  {truth(data?.LatestDiagnosis) ? (
                    <> View diagnosis history </>
                  ) : (
                    <> Start health check </>
                  )}
                </a>
                {truth(data?.ProduceDemandPreview) ? (
                  <>
                    <div className="market-insights-heading">
                      <strong> Top market insights </strong>
                      <a href="/market-insights"> View all </a>
                    </div>
                    {truth(data?.ProduceDemandPreview)
                      ? items(data?.ProduceDemandPreview).map(
                          (item13, index13) => (
                            <Fragment key={index13}>
                              <div className="produce-demand-summary">
                                <strong>{text(item13?.ProduceType)}</strong> ·{" "}
                                {text(item13?.Activity)}
                                <span>
                                  {" "}
                                  Last 30 days · {text(item13?.Views)} views ·{" "}
                                  {text(item13?.CartAdds)} cart adds ·{" "}
                                  {text(item13?.Negotiations)} negotiations ·{" "}
                                  {text(item13?.Orders)} orders{" "}
                                </span>
                                {truth(item13?.PriceAvailable) ? (
                                  <>
                                    <span>
                                      {" "}
                                      Local prices: ₦
                                      {text(money0(item13?.LocalPriceMin))}–₦
                                      {text(money0(item13?.LocalPriceMax))} ·
                                      median ₦
                                      {text(money0(item13?.LocalPriceMedian))}
                                    </span>
                                  </>
                                ) : null}
                                {truth(item13?.PricePosition) ? (
                                  <>
                                    <span>
                                      {" "}
                                      Your average price: ₦
                                      {text(money0(item13?.FarmerPrice))} ·{" "}
                                      {text(item13?.PricePosition)} (
                                      {text(
                                        printf(
                                          "%+.0f%%",
                                          item13?.PriceDifferencePercent,
                                        ),
                                      )}
                                      ){" "}
                                    </span>
                                  </>
                                ) : null}
                                {truth(item13?.StateListings) ? (
                                  <>
                                    <span>
                                      {" "}
                                      State comparison:{" "}
                                      {text(item13?.StateListings)} listings ·
                                      median ₦
                                      {text(money0(item13?.StatePriceMedian))}
                                    </span>
                                  </>
                                ) : null}
                                {truth(item13?.NearbyListings) ? (
                                  <>
                                    <span>
                                      {" "}
                                      Nearby LGA comparison:{" "}
                                      {text(item13?.NearbyListings)} listings ·
                                      median ₦
                                      {text(money0(item13?.NearbyPriceMedian))}
                                    </span>
                                  </>
                                ) : null}
                                {truth(item13?.NationalListings) ? (
                                  <>
                                    <span>
                                      {" "}
                                      National comparison:{" "}
                                      {text(item13?.NationalListings)} listings
                                      · median ₦
                                      {text(
                                        money0(item13?.NationalPriceMedian),
                                      )}
                                    </span>
                                  </>
                                ) : null}
                                {truth(item13?.ExternalPriceAvailable) ? (
                                  <>
                                    <span>
                                      {" "}
                                      Verified external observations:{" "}
                                      {text(item13?.ExternalListings)} · median
                                      ₦
                                      {text(
                                        money0(item13?.ExternalPriceMedian),
                                      )}
                                    </span>
                                  </>
                                ) : null}
                                <small>{text(item13?.Recommendation)}</small>
                              </div>
                            </Fragment>
                          ),
                        )
                      : null}
                  </>
                ) : null}
              </section>
              <section
                className="panel market-opportunity"
                aria-labelledby="market-opportunity-title"
              >
                <div className="section-heading">
                  <div>
                    <p className="eyebrow"> Marketplace </p>
                    <h2 id="market-opportunity-title"> Market opportunity </h2>
                  </div>
                  <a href="/marketplace"> Marketplace </a>
                </div>
                <div className="opportunity-content">
                  <div className="opportunity-icon" aria-hidden="true">
                    {" "}
                    ↗{" "}
                  </div>
                  <div>
                    <span className="opportunity-label">
                      {truth(data?.Demand?.ProduceType) ? (
                        <> Your product demand signal </>
                      ) : (
                        <> Platform activity </>
                      )}
                    </span>
                    <h3>
                      {truth(data?.Demand?.ProduceType) ? (
                        <>{text(data?.Demand?.Activity)}</>
                      ) : truth(
                          truth(data?.MarketOrders30) ||
                            truth(data?.MarketNegotiations30),
                        ) ? (
                        <> Buyer activity detected in the last 30 days </>
                      ) : (
                        <> No buyer activity recorded in the last 30 days </>
                      )}
                    </h3>
                    <p>
                      {truth(data?.Demand?.ProduceType) ? (
                        <>
                          {" "}
                          Based on recorded activity across your product
                          listings in the last 30 days.{" "}
                        </>
                      ) : (
                        <>
                          {" "}
                          Based on recorded Agrilinks Nigeria orders and
                          negotiations in the last 30 days; this is not a
                          market-wide forecast.{" "}
                        </>
                      )}
                    </p>
                  </div>
                </div>
                <div className="opportunity-stats">
                  <div>
                    <span> Your active listings </span>
                    <strong>{text(data?.MarketListings)}</strong>
                  </div>
                  <div>
                    <span> Your listings with photos </span>
                    <strong>
                      {text(data?.ListingsWithPhotos)}/
                      {text(data?.ListingsActive)}
                    </strong>
                  </div>
                  <div>
                    <span> Platform price range </span>
                    <strong>
                      {truth(data?.MarketHasPrices) ? (
                        <>
                          {" "}
                          ₦{text(money0(data?.MarketPriceMin))}–₦
                          {text(money0(data?.MarketPriceMax))}
                        </>
                      ) : (
                        <> Unavailable </>
                      )}
                    </strong>
                  </div>
                  <div>
                    <span> Typical price </span>
                    <strong>
                      {truth(data?.MarketHasPrices) ? (
                        <> ₦{text(money0(data?.MarketPriceMedian))}</>
                      ) : (
                        <> — </>
                      )}
                    </strong>
                  </div>
                  <div>
                    <span> Your orders · last 30 days </span>
                    <strong>{text(data?.MarketOrders30)}</strong>
                  </div>
                  <div>
                    <span> Your negotiations · last 30 days </span>
                    <strong>{text(data?.MarketNegotiations30)}</strong>
                  </div>
                  {truth(data?.Demand?.ProduceType) ? (
                    <>
                      <div>
                        <span> Views · 30 days </span>
                        <strong>{text(data?.Demand?.Views)}</strong>
                      </div>
                      <div>
                        <span> Times added to cart · 30 days </span>
                        <strong>{text(data?.Demand?.CartAdds)}</strong>
                      </div>
                      <div>
                        <span> How much was ordered · 30 days </span>
                        <strong>
                          {text(printf("%.0f", data?.Demand?.UnitsOrdered))}
                        </strong>
                      </div>
                      <div>
                        <span> Trend vs previous 30 days </span>
                        <strong>
                          {text(printf("%+.0f%%", data?.Demand?.TrendPercent))}
                        </strong>
                      </div>
                      <div>
                        <span>
                          {" "}
                          Platform searches ·{" "}
                          {text(data?.Demand?.ComparisonLocation)}
                        </span>
                        <strong>{text(data?.Demand?.Searches)}</strong>
                      </div>
                      <div>
                        <span> Active negotiations </span>
                        <strong>
                          {text(data?.Demand?.NegotiationsActive)}
                        </strong>
                      </div>
                      <div>
                        <span> Accepted negotiations </span>
                        <strong>
                          {text(data?.Demand?.NegotiationsAccepted)}
                        </strong>
                      </div>
                      <div>
                        <span> Rejected negotiations </span>
                        <strong>
                          {text(data?.Demand?.NegotiationsRejected)}
                        </strong>
                      </div>
                      <div>
                        <span> Negotiation conversion </span>
                        <strong>
                          {truth(
                            truth(data?.Demand?.NegotiationsActive) ||
                              truth(data?.Demand?.NegotiationsAccepted) ||
                              truth(data?.Demand?.NegotiationsRejected),
                          ) ? (
                            <>
                              {text(
                                printf(
                                  "%.0f%%",
                                  data?.Demand?.NegotiationConversionRate,
                                ),
                              )}
                            </>
                          ) : (
                            <> Awaiting activity </>
                          )}
                        </strong>
                      </div>
                      <div>
                        <span> Demand score </span>
                        <strong>
                          {truth(
                            truth(data?.Demand?.Views) ||
                              truth(data?.Demand?.CartAdds) ||
                              truth(data?.Demand?.Orders) ||
                              truth(data?.Demand?.Negotiations),
                          ) ? (
                            <>
                              {text(
                                printf("%.0f/100", data?.Demand?.DemandScore),
                              )}
                            </>
                          ) : (
                            <> Awaiting activity </>
                          )}
                        </strong>
                      </div>
                      <div>
                        <span> Typical offer price </span>
                        <strong>
                          {truth(data?.Demand?.AverageOfferPrice) ? (
                            <>
                              {" "}
                              ₦{text(money0(data?.Demand?.AverageOfferPrice))}
                            </>
                          ) : (
                            <> — </>
                          )}
                        </strong>
                      </div>
                      <div>
                        <span> How much was negotiated · 30 days </span>
                        <strong>
                          {text(printf("%.0f", data?.Demand?.NegotiatedUnits))}
                        </strong>
                      </div>
                      <div>
                        <span> Share sold · 30 days </span>
                        <strong>
                          {truth(data?.Demand?.LifecycleDataAvailable) ? (
                            <>
                              {text(
                                printf("%.0f%%", data?.Demand?.SellThroughRate),
                              )}
                            </>
                          ) : (
                            <> Awaiting sales data </>
                          )}
                        </strong>
                      </div>
                    </>
                  ) : null}
                  {truth(data?.Demand?.TimeToFirstOrderHours) ? (
                    <>
                      <div>
                        <span> Average time to first order </span>
                        <strong>
                          {text(
                            printf(
                              "%.0f hours",
                              data?.Demand?.TimeToFirstOrderHours,
                            ),
                          )}
                        </strong>
                      </div>
                    </>
                  ) : null}
                  {truth(data?.Demand?.TimeToSellOutHours) ? (
                    <>
                      <div>
                        <span> Average time to sell out </span>
                        <strong>
                          {text(
                            printf(
                              "%.0f hours",
                              data?.Demand?.TimeToSellOutHours,
                            ),
                          )}
                        </strong>
                      </div>
                    </>
                  ) : null}
                </div>
                <a href="/storage" className="panel-action">
                  {" "}
                  Manage listings in Storage{" "}
                </a>
              </section>
            </section>
          </>
        ) : null}
        {truth(!truth(data?.IsBuyer)) ? (
          <>
            <section
              className="panel storage-overview"
              aria-labelledby="storage-overview-title"
              hidden={true}
            >
              <div className="section-heading">
                <div>
                  <p className="eyebrow"> Products </p>
                  <h2 id="storage-overview-title"> Your products </h2>
                </div>
                <a href="/storage"> Manage products </a>
              </div>
              <div className="produce-list">
                {truth(data?.StorageCrops) ? (
                  items(data?.StorageCrops).map((item14, index14) => (
                    <Fragment key={index14}>
                      <div className="produce-row">
                        <div className="produce-info">
                          <span className="produce-icon" aria-hidden="true">
                            {truth(item14?.Status === "active") ? (
                              <> 🌾 </>
                            ) : (
                              <> 📦 </>
                            )}
                          </span>
                          <div>
                            <h3>{text(item14?.Name)}</h3>
                            <span>
                              {truth(item14?.Status === "active") ? (
                                <> Listed for sale </>
                              ) : (
                                <> Not listed </>
                              )}
                            </span>
                          </div>
                        </div>
                        <div className="produce-quantity">
                          <strong>
                            {text(printf("%.0f", item14?.Quantity))}
                          </strong>
                          <span>{text(item14?.Unit)}</span>
                        </div>
                        <span className="stock-status healthy">
                          {truth(item14?.Quantity === 0.0) ? (
                            <> Empty </>
                          ) : truth(item14?.Status === "active") ? (
                            <> Listed </>
                          ) : (
                            <> Stored </>
                          )}
                        </span>
                      </div>
                    </Fragment>
                  ))
                ) : (
                  <>
                    <div className="empty-state">
                      <p> No products in storage yet. </p>
                      <a href="/storage"> Add your first product </a>
                    </div>
                  </>
                )}
              </div>
            </section>
          </>
        ) : null}
        <section className="dashboard-grid">
          <section
            className="panel priorities"
            aria-labelledby="priorities-title"
          >
            <div className="section-heading">
              <div>
                <p className="eyebrow"> Focus </p>
                <h2 id="priorities-title"> Today's priorities </h2>
              </div>
            </div>
            {truth(data?.Priorities) ? (
              <>
                {truth(data?.Priorities)
                  ? items(data?.Priorities).map((item15, index15) => (
                      <Fragment key={index15}>
                        <article
                          className={"priority priority-" + text(item15?.Tone)}
                        >
                          <div>
                            <span className="priority-label">
                              {text(item15?.Label)}
                            </span>
                            <h3>{text(item15?.Title)}</h3>
                            <p>{text(item15?.Detail)}</p>
                          </div>
                          <a href={text(item15?.Href)}>
                            {text(item15?.Action)}
                          </a>
                        </article>
                      </Fragment>
                    ))
                  : null}
              </>
            ) : (
              <>
                <article className="priority priority-success">
                  <div>
                    <span className="priority-label"> On track </span>
                    <h3> Your account has no urgent actions </h3>
                    <p>
                      {" "}
                      Use the marketplace and your account tools to keep
                      momentum going.{" "}
                    </p>
                  </div>
                </article>
              </>
            )}
          </section>
        </section>
        <section
          className="panel activity"
          aria-labelledby="recent-activity-title"
        >
          <div className="section-heading">
            <div>
              <p className="eyebrow"> Latest </p>
              <h2 id="recent-activity-title"> Recent activity </h2>
            </div>
            <a href={truth(data?.IsBuyer) ? "/profile" : "/sales"}>
              {" "}
              View all{" "}
            </a>
          </div>
          {truth(data?.RecentOrders) ? (
            <>
              <div className="activity-list">
                {truth(data?.RecentOrders)
                  ? items(data?.RecentOrders).map((item16, index16) => (
                      <Fragment key={index16}>
                        <article className="activity-row">
                          <div
                            className="activity-mark"
                            aria-hidden="true"
                          ></div>
                          <div className="activity-main">
                            <h3>{text(item16?.ProductName)}</h3>
                            <p>
                              {truth(data?.IsBuyer) ? (
                                <> From {text(item16?.SellerName)}</>
                              ) : (
                                <> Purchased by {text(item16?.BuyerName)}</>
                              )}{" "}
                              · {text(printf("%.0f", item16?.Quantity))}
                              {text(item16?.ProductUnit)}
                            </p>
                          </div>
                          <div className="activity-value">
                            <strong> ₦{text(money(item16?.TotalPrice))}</strong>
                            <span>
                              {text(humanOrderStatus(item16?.Status))}
                            </span>
                          </div>
                        </article>
                      </Fragment>
                    ))
                  : null}
              </div>
            </>
          ) : (
            <>
              <div className="empty-state">
                <h3> No activity yet </h3>
                <p>
                  {truth(data?.IsBuyer) ? (
                    <> Your completed purchases will appear here. </>
                  ) : (
                    <>
                      {" "}
                      Add a product, publish a listing, and complete your first
                      sale to see performance here.{" "}
                    </>
                  )}
                </p>
              </div>
            </>
          )}
        </section>
        <section
          className="trust-strip"
          aria-label="Agrilinks Nigeria platform features"
        >
          <div className="trust-item">
            <span className="trust-icon" aria-hidden="true">
              {" "}
              ✓{" "}
            </span>
            <div>
              <strong> Verified marketplace </strong>
              <span> Built for trusted agricultural trade </span>
            </div>
          </div>
          <div className="trust-item">
            <span className="trust-icon" aria-hidden="true">
              {" "}
              AI{" "}
            </span>
            <div>
              <strong> AI-assisted farm health </strong>
              <span> Get insights before problems grow </span>
            </div>
          </div>
          <div className="trust-item">
            <span className="trust-icon" aria-hidden="true">
              {" "}
              🔒{" "}
            </span>
            <div>
              <strong> Secure transactions </strong>
              <span> Your marketplace activity is protected </span>
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}
