import Workspace, { WorkspaceHeading } from "../components/Workspace.jsx";
import { Select } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text, humanStatus } from "../lib/view.js";
import mountDeliveryPartner from "../behaviors/delivery_partner.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function DeliveryPartnerPage({ data }) {
  usePageBehaviors([mountDeliveryPartner], data);
  return (
    <Workspace current="/delivery-partner">
      <main className="dashboard account-page">
        <WorkspaceHeading
          eyebrow="DELIVERY WORKSPACE"
          title="Delivery partner"
          description="Manage your delivery business and keep each job moving."
        />
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
        {truth(data?.PartnerAccounts) ? (
          <>
            <section className="panel">
              <h2>Your delivery business</h2>
              {truth(data?.PartnerAccounts)
                ? items(data?.PartnerAccounts).map((item1, index1) => (
                    <Fragment key={index1}>
                      <article>
                        <h3>{text(item1?.Name)}</h3>
                        <p>Phone: {text(item1?.Phone)}</p>
                        {truth(item1?.Email) ? (
                          <>
                            <p>Email: {text(item1?.Email)}</p>
                          </>
                        ) : null}
                        {truth(item1?.VehicleType) ? (
                          <>
                            <p>Vehicle: {text(item1?.VehicleType)}</p>
                          </>
                        ) : null}
                        <p>Status: {text(humanStatus(item1?.Status))}</p>
                        <p>
                          {" "}
                          Approval:{" "}
                          {truth(item1?.Verified) ? (
                            <>Approved</>
                          ) : (
                            <>Waiting for approval</>
                          )}
                        </p>
                        <p>Your access: {text(item1?.MemberRole)}</p>
                      </article>
                    </Fragment>
                  ))
                : null}
            </section>
          </>
        ) : null}
        <section className="panel">
          <h2>Register a delivery business</h2>
          <p>
            {" "}
            Registering lets you receive delivery jobs, update their progress,
            and share locations with buyers.{" "}
          </p>
          <form method="POST" action="/delivery-partner">
            <label>
              {" "}
              Business or driver name{" "}
              <input
                type="text"
                name="name"
                defaultValue={text(data?.User?.FullName)}
                required={true}
              />
            </label>
            <label>
              {" "}
              Phone number{" "}
              <input
                type="tel"
                name="phone"
                defaultValue={text(data?.User?.Phone)}
                required={true}
              />
            </label>
            <label>
              {" "}
              Email address{" "}
              <input
                type="email"
                name="email"
                defaultValue={text(data?.User?.Email)}
              />
            </label>
            <label>
              {" "}
              State{" "}
              <Select name="state_id" id="partner-state">
                <option value="">Select your state</option>
              </Select>
            </label>
            <label>
              {" "}
              Local Government Area (LGA){" "}
              <Select name="lga_id" id="partner-lga" disabled={true}>
                <option value="">Select your LGA</option>
              </Select>
            </label>
            <label>
              {" "}
              Vehicle type{" "}
              <input
                type="text"
                name="vehicle_type"
                placeholder="For example: motorcycle, pickup, truck"
              />
            </label>
            <label>
              {" "}
              Vehicle registration number{" "}
              <input type="text" name="registration_number" />
            </label>
            <button type="submit">Register delivery business</button>
          </form>
        </section>
        {truth(data?.PartnersForReview) ? (
          <>
            <section className="panel">
              <h2>Delivery partners waiting for review</h2>
              <div
                className="table-scroll"
                role="region"
                aria-label="Delivery Partner table"
                tabIndex="0"
              >
                <table>
                  <thead>
                    <tr>
                      <th>Business</th>
                      <th>Driver</th>
                      <th>Vehicle</th>
                      <th>Approval</th>
                      <th>Status</th>
                      <th>Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {truth(data?.PartnersForReview)
                      ? items(data?.PartnersForReview).map((item2, index2) => (
                          <Fragment key={index2}>
                            <tr>
                              <td>{text(item2?.Name)}</td>
                              <td>{text(item2?.Phone)}</td>
                              <td>
                                {truth(item2?.VehicleType) ? (
                                  <>{text(item2?.VehicleType)}</>
                                ) : (
                                  <>Not given</>
                                )}
                              </td>
                              <td>
                                {truth(item2?.Verified) ? (
                                  <>Approved</>
                                ) : (
                                  <>Waiting</>
                                )}
                              </td>
                              <td>{text(humanStatus(item2?.Status))}</td>
                              <td>
                                <form
                                  method="POST"
                                  action="/delivery-partner/review"
                                >
                                  <input
                                    type="hidden"
                                    name="partner_id"
                                    defaultValue={text(item2?.ID)}
                                  />
                                  <Select name="action" required={true}>
                                    <option value="approve">Approve</option>
                                    <option value="unapprove">
                                      Remove approval
                                    </option>
                                    <option value="activate">
                                      Mark active
                                    </option>
                                    <option value="pause">Pause</option>
                                    <option value="suspend">Suspend</option>
                                  </Select>
                                  <button type="submit">Save</button>
                                </form>
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
      </main>
    </Workspace>
  );
}
