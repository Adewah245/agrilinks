import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import {
  truth,
  items,
  text,
  humanOrderStatus,
  humanStatus,
  formatDate,
} from "../lib/view.js";
export default function DeliveriesPage({ data }) {
  return (
    <>
      <main className="dashboard account-page">
        <p>
          <a href="/dashboard">← Dashboard</a>
        </p>
        <h1>Deliveries</h1>
        {truth(data?.Error) ? (
          <>
            <p role="alert">{text(data?.Error)}</p>
          </>
        ) : null}
        {truth(data?.Notice) ? (
          <>
            <p role="status">{text(data?.Notice)}</p>
          </>
        ) : null}
        <p>
          <a href="/delivery-partner">Delivery partner account</a>
        </p>
        {truth(data?.EligibleOrders) ? (
          <>
            <section className="panel">
              <h2>Arrange delivery</h2>
              <p>These orders are paid. Tell us where each one should go.</p>
              {truth(data?.EligibleOrders)
                ? items(data?.EligibleOrders).map((item1, index1) => (
                    <Fragment key={index1}>
                      <form method="POST" action="/deliveries/request">
                        <h3>
                          Order #{text(item1?.ID)} — {text(item1?.ProductNames)}
                        </h3>
                        <p>
                          Seller: {text(item1?.SellerName)} ·{" "}
                          {text(humanOrderStatus(item1?.Status))}
                        </p>
                        <input
                          type="hidden"
                          name="order_id"
                          defaultValue={text(item1?.ID)}
                        />
                        <label>
                          {" "}
                          Delivery address{" "}
                          <input
                            type="text"
                            name="destination_address"
                            placeholder="Street, community, town or landmark"
                            required={true}
                          />
                        </label>
                        <button type="submit">Request delivery</button>
                      </form>
                    </Fragment>
                  ))
                : null}
            </section>
          </>
        ) : null}
        {truth(data?.Deliveries) ? (
          <>
            <section className="panel">
              <h2>My deliveries</h2>
              {truth(data?.Deliveries)
                ? items(data?.Deliveries).map((item2, index2) => (
                    <Fragment key={index2}>
                      <article>
                        <h3>
                          Order #{text(item2?.OrderID)} —{" "}
                          {text(humanStatus(item2?.Status))}
                        </h3>
                        <p>From: {text(item2?.PickupAddress)}</p>
                        <p>To: {text(item2?.DestinationAddress)}</p>
                        {truth(item2?.PartnerName) ? (
                          <>
                            <p>Delivery partner: {text(item2?.PartnerName)}</p>
                          </>
                        ) : null}
                        {truth(item2?.TrackingCode) ? (
                          <>
                            <p>Tracking code: {text(item2?.TrackingCode)}</p>
                          </>
                        ) : null}
                        <p>
                          <a href={"/deliveries?id=" + text(item2?.ID)}>
                            View tracking
                          </a>
                        </p>
                        {truth(item2?.Status === "pending") ? (
                          <>
                            <form method="POST" action="/deliveries/status">
                              <input
                                type="hidden"
                                name="delivery_id"
                                defaultValue={text(item2?.ID)}
                              />
                              <input
                                type="hidden"
                                name="status"
                                defaultValue="cancelled"
                              />
                              <button type="submit">
                                Cancel delivery request
                              </button>
                            </form>
                          </>
                        ) : null}
                      </article>
                    </Fragment>
                  ))
                : null}
            </section>
          </>
        ) : (
          <>
            {truth(data?.User?.IsBuyer) ? (
              <>
                <p>No deliveries have been arranged yet.</p>
              </>
            ) : null}
          </>
        )}
        {truth(data?.SellerDeliveries) ? (
          <>
            <section className="panel">
              <h2>Deliveries for your sales</h2>
              {truth(data?.SellerDeliveries)
                ? items(data?.SellerDeliveries).map((item3, index3) => (
                    <Fragment key={index3}>
                      <article>
                        <h3>
                          Order #{text(item3?.OrderID)} —{" "}
                          {text(humanStatus(item3?.Status))}
                        </h3>
                        <p>Buyer: {text(item3?.BuyerName)}</p>
                        <p>Pick up from: {text(item3?.PickupAddress)}</p>
                        <p>Deliver to: {text(item3?.DestinationAddress)}</p>
                        {truth(item3?.PartnerName) ? (
                          <>
                            <p>Delivery partner: {text(item3?.PartnerName)}</p>
                          </>
                        ) : null}
                        {truth(
                          truth(item3?.Status === "pending") ||
                            truth(item3?.Status === "assigned"),
                        ) ? (
                          <>
                            {truth(data?.AvailablePartners) ? (
                              <>
                                <form method="POST" action="/deliveries/assign">
                                  <input
                                    type="hidden"
                                    name="delivery_id"
                                    defaultValue={text(item3?.ID)}
                                  />
                                  <label>
                                    {" "}
                                    Delivery partner{" "}
                                    <Select name="partner_id" required={true}>
                                      <option value="">
                                        Choose an approved partner
                                      </option>
                                      {truth(data?.AvailablePartners)
                                        ? items(data?.AvailablePartners).map(
                                            (item4, index4) => (
                                              <Fragment key={index4}>
                                                <option value={text(item4?.ID)}>
                                                  {text(item4?.Name)}
                                                  {truth(item4?.VehicleType) ? (
                                                    <>
                                                      {" "}
                                                      —{" "}
                                                      {text(item4?.VehicleType)}
                                                    </>
                                                  ) : null}
                                                </option>
                                              </Fragment>
                                            ),
                                          )
                                        : null}
                                    </Select>
                                  </label>
                                  <button type="submit">Assign partner</button>
                                </form>
                              </>
                            ) : (
                              <>
                                <p>
                                  No approved delivery partners are available
                                  yet.
                                </p>
                              </>
                            )}
                            <form method="POST" action="/deliveries/status">
                              <input
                                type="hidden"
                                name="delivery_id"
                                defaultValue={text(item3?.ID)}
                              />
                              <input
                                type="hidden"
                                name="status"
                                defaultValue="cancelled"
                              />
                              <button type="submit">
                                Cancel this delivery
                              </button>
                            </form>
                          </>
                        ) : null}
                      </article>
                    </Fragment>
                  ))
                : null}
            </section>
          </>
        ) : (
          <>
            {truth(!truth(data?.User?.IsBuyer)) ? (
              <>
                <p>No deliveries have been requested for your sales yet.</p>
              </>
            ) : null}
          </>
        )}
        {truth(data?.PartnerDeliveries) ? (
          <>
            <section className="panel">
              <h2>Deliveries assigned to you</h2>
              {truth(data?.PartnerDeliveries)
                ? items(data?.PartnerDeliveries).map((item5, index5) => (
                    <Fragment key={index5}>
                      <article>
                        <h3>
                          Order #{text(item5?.OrderID)} —{" "}
                          {text(humanStatus(item5?.Status))}
                        </h3>
                        <p>Buyer: {text(item5?.BuyerName)}</p>
                        <p>Pick up from: {text(item5?.PickupAddress)}</p>
                        <p>Deliver to: {text(item5?.DestinationAddress)}</p>
                        {truth(item5?.TrackingCode) ? (
                          <>
                            <p>Tracking code: {text(item5?.TrackingCode)}</p>
                          </>
                        ) : null}
                        {truth(item5?.Status === "assigned") ? (
                          <>
                            <form method="POST" action="/deliveries/status">
                              <input
                                type="hidden"
                                name="delivery_id"
                                defaultValue={text(item5?.ID)}
                              />
                              <input
                                type="hidden"
                                name="status"
                                defaultValue="picked_up"
                              />
                              <button type="submit">Mark as picked up</button>
                            </form>
                          </>
                        ) : truth(item5?.Status === "picked_up") ? (
                          <>
                            <form method="POST" action="/deliveries/status">
                              <input
                                type="hidden"
                                name="delivery_id"
                                defaultValue={text(item5?.ID)}
                              />
                              <input
                                type="hidden"
                                name="status"
                                defaultValue="in_transit"
                              />
                              <button type="submit">Mark as on the way</button>
                            </form>
                          </>
                        ) : truth(item5?.Status === "in_transit") ? (
                          <>
                            <form method="POST" action="/deliveries/status">
                              <input
                                type="hidden"
                                name="delivery_id"
                                defaultValue={text(item5?.ID)}
                              />
                              <input
                                type="hidden"
                                name="status"
                                defaultValue="delivered"
                              />
                              <button type="submit">Mark as delivered</button>
                            </form>
                          </>
                        ) : null}
                        {truth(
                          truth(item5?.Status === "assigned") ||
                            truth(item5?.Status === "picked_up") ||
                            truth(item5?.Status === "in_transit"),
                        ) ? (
                          <>
                            <form method="POST" action="/deliveries/tracking">
                              <h4>Add a location update</h4>
                              <input
                                type="hidden"
                                name="delivery_id"
                                defaultValue={text(item5?.ID)}
                              />
                              <label>
                                {" "}
                                Latitude{" "}
                                <input
                                  type="number"
                                  name="latitude"
                                  step="any"
                                  min="-90"
                                  max="90"
                                  required={true}
                                />
                              </label>
                              <label>
                                {" "}
                                Longitude{" "}
                                <input
                                  type="number"
                                  name="longitude"
                                  step="any"
                                  min="-180"
                                  max="180"
                                  required={true}
                                />
                              </label>
                              <label>
                                {" "}
                                Place or landmark{" "}
                                <input
                                  type="text"
                                  name="location_name"
                                  placeholder="For example: Kano central market"
                                />
                              </label>
                              <label>
                                {" "}
                                Note for the buyer{" "}
                                <Textarea name="note" rows="2"></Textarea>
                              </label>
                              <button type="submit">Add tracking update</button>
                            </form>
                          </>
                        ) : null}
                      </article>
                    </Fragment>
                  ))
                : null}
            </section>
          </>
        ) : null}
        {truth(data?.AllDeliveries) ? (
          <>
            <section className="panel">
              <h2>All deliveries</h2>
              <div
                className="table-scroll"
                role="region"
                aria-label="Deliveries table"
                tabIndex="0"
              >
                <table>
                  <thead>
                    <tr>
                      <th>Order</th>
                      <th>Buyer</th>
                      <th>Partner</th>
                      <th>Status</th>
                      <th>Tracking</th>
                    </tr>
                  </thead>
                  <tbody>
                    {truth(data?.AllDeliveries)
                      ? items(data?.AllDeliveries).map((item6, index6) => (
                          <Fragment key={index6}>
                            <tr>
                              <td>#{text(item6?.OrderID)}</td>
                              <td>{text(item6?.BuyerName)}</td>
                              <td>
                                {truth(item6?.PartnerName) ? (
                                  <>{text(item6?.PartnerName)}</>
                                ) : (
                                  <>Not assigned</>
                                )}
                              </td>
                              <td>{text(humanStatus(item6?.Status))}</td>
                              <td>
                                <a href={"/deliveries?id=" + text(item6?.ID)}>
                                  View
                                </a>
                              </td>
                            </tr>
                          </Fragment>
                        ))
                      : null}
                  </tbody>
                </table>
              </div>
            </section>
          </>
        ) : null}
        {truth(data?.SelectedID) ? (
          <>
            <section className="panel">
              <h2>Tracking updates</h2>
              {truth(data?.Tracking) ? (
                <>
                  <ol>
                    {truth(data?.Tracking)
                      ? items(data?.Tracking).map((item7, index7) => (
                          <Fragment key={index7}>
                            <li>
                              {text(
                                formatDate(
                                  item7?.CreatedAt,
                                  "2 Jan 2006 3:04 PM",
                                ),
                              )}{" "}
                              —{" "}
                              {truth(item7?.LocationName) ? (
                                <>{text(item7?.LocationName)}</>
                              ) : (
                                <>Location shared</>
                              )}
                              {truth(item7?.Note) ? (
                                <>: {text(item7?.Note)}</>
                              ) : null}
                            </li>
                          </Fragment>
                        ))
                      : null}
                  </ol>
                </>
              ) : (
                <>
                  <p>No tracking updates yet.</p>
                </>
              )}
            </section>
          </>
        ) : null}
      </main>
    </>
  );
}
