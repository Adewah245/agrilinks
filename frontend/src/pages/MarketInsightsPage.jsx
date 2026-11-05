import { WorkspaceSidebar } from "../components/Workspace.jsx";
import { Fragment } from "react";
import { truth, items, text, money0 } from "../lib/view.js";
export default function MarketInsightsPage({ data }) {
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current="/market-insights">
        <a href="/dashboard"> Dashboard </a>
        <a href="/storage"> Storage </a>
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
      </WorkspaceSidebar>
      <main className="dashboard">
        <section
          className="dashboard-header"
          aria-labelledby="market-insights-title"
        >
          <div className="dashboard-intro">
            <p className="eyebrow"> Market Intelligence </p>
            <h1 id="market-insights-title"> Agricultural Product Insights </h1>
            <p>
              {" "}
              Buyer activity and independent price comparisons for{" "}
              {text(data?.FullName)}.{" "}
            </p>
          </div>
        </section>
        <section className="panel" aria-label="Market insights">
          {truth(data?.Summaries) ? (
            items(data?.Summaries).map((item1, index1) => (
              <Fragment key={index1}>
                <article className="produce-demand-summary">
                  <strong>{text(item1?.ProduceType)}</strong>
                  <span>
                    {text(item1?.Activity)} · {text(item1?.Views)} views ·{" "}
                    {text(item1?.CartAdds)} cart adds · {text(item1?.Orders)}{" "}
                    orders{" "}
                  </span>
                  <span>
                    {" "}
                    Your average price: ₦{text(money0(item1?.FarmerPrice))}
                  </span>
                  {truth(item1?.PriceAvailable) ? (
                    <>
                      <span>
                        {" "}
                        Independent local median: ₦
                        {text(money0(item1?.LocalPriceMedian))}
                      </span>
                    </>
                  ) : (
                    <>
                      <span>
                        {" "}
                        No independent local price comparison is available
                        yet.{" "}
                      </span>
                    </>
                  )}
                  {truth(item1?.StateListings) ? (
                    <>
                      <span>
                        {" "}
                        State: {text(item1?.StateListings)} other listings ·
                        median ₦{text(money0(item1?.StatePriceMedian))}
                      </span>
                    </>
                  ) : null}
                  {truth(item1?.NearbyListings) ? (
                    <>
                      <span>
                        {" "}
                        Nearby LGAs: {text(item1?.NearbyListings)} other
                        listings · median ₦
                        {text(money0(item1?.NearbyPriceMedian))}
                      </span>
                    </>
                  ) : null}
                  {truth(item1?.NationalListings) ? (
                    <>
                      <span>
                        {" "}
                        National: {text(item1?.NationalListings)} other listings
                        · median ₦{text(money0(item1?.NationalPriceMedian))}
                      </span>
                    </>
                  ) : null}
                  <small>{text(item1?.Recommendation)}</small>
                </article>
              </Fragment>
            ))
          ) : (
            <>
              <div className="empty-state">
                <p>
                  {" "}
                  Add agricultural products to storage to begin receiving market
                  insights.{" "}
                </p>
                <a href="/storage" className="primary-btn">
                  {" "}
                  Go to Storage{" "}
                </a>
              </div>
            </>
          )}
        </section>
      </main>
    </div>
  );
}
