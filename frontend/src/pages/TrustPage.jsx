import Workspace, { WorkspaceHeading } from "../components/Workspace.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text, humanBadge, formatDate } from "../lib/view.js";
export default function TrustPage({ data }) {
  return (
    <Workspace current="/trust" staff>
      <main className="dashboard account-page">
        <WorkspaceHeading
          eyebrow="STAFF WORKSPACE"
          title="Trust badges"
          description="Help buyers find verified farmers and trusted sellers. Identity documents and NIN details stay private."
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
        <section className="panel">
          <h2>Issue a badge</h2>
          <form method="POST" action="/trust">
            <input type="hidden" name="action" defaultValue="issue" />
            <label>
              {" "}
              Who is this badge for?{" "}
              <Select name="owner_type" required={true}>
                <option
                  value="user"
                  selected={truth(data?.OwnerType === "user")}
                >
                  User
                </option>
                <option
                  value="organization"
                  selected={truth(data?.OwnerType === "organization")}
                >
                  Organization
                </option>
              </Select>
            </label>
            <label>
              {" "}
              Their account ID{" "}
              <input
                type="number"
                name="owner_id"
                min="1"
                defaultValue={
                  truth(data?.OwnerID > 0) ? text(data?.OwnerID) : ""
                }
                required={true}
              />
            </label>
            <label>
              {" "}
              Badge{" "}
              <Select name="badge_type" required={true}>
                <option value="verified_farmer">Verified farmer</option>
                <option value="verified_organization">
                  Verified organization
                </option>
                <option value="government_partner">Government partner</option>
                <option value="trusted_seller">Trusted seller</option>
              </Select>
            </label>
            <label>
              {" "}
              Public note{" "}
              <Textarea
                name="description"
                rows="2"
                placeholder="Optional explanation shown with the badge"
              ></Textarea>
            </label>
            <label>
              {" "}
              Expires on <input type="date" name="expires_at" />
            </label>
            <button type="submit">Issue badge</button>
          </form>
        </section>
        <section className="panel">
          <h2>Check an account</h2>
          <form method="GET" action="/trust">
            <label>
              {" "}
              Account type{" "}
              <Select name="owner_type">
                <option
                  value="user"
                  selected={truth(data?.OwnerType === "user")}
                >
                  User
                </option>
                <option
                  value="organization"
                  selected={truth(data?.OwnerType === "organization")}
                >
                  Organization
                </option>
              </Select>
            </label>
            <label>
              {" "}
              Account ID{" "}
              <input
                type="number"
                name="owner_id"
                min="1"
                defaultValue={
                  truth(data?.OwnerID > 0) ? text(data?.OwnerID) : ""
                }
                required={true}
              />
            </label>
            <button type="submit">Show badges</button>
          </form>
          {truth(data?.Badges) ? (
            <>
              <div
                className="table-scroll"
                role="region"
                aria-label="Trust Badges table"
                tabIndex="0"
              >
                <table>
                  <thead>
                    <tr>
                      <th>Badge</th>
                      <th>Note</th>
                      <th>Issued</th>
                      <th>Expires</th>
                      <th>Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {truth(data?.Badges)
                      ? items(data?.Badges).map((item1, index1) => (
                          <Fragment key={index1}>
                            <tr>
                              <td>{text(humanBadge(item1?.BadgeType))}</td>
                              <td>
                                {truth(item1?.Description) ? (
                                  <>{text(item1?.Description)}</>
                                ) : (
                                  <>—</>
                                )}
                              </td>
                              <td>
                                {text(
                                  formatDate(item1?.IssuedAt, "02 Jan 2006"),
                                )}
                              </td>
                              <td>
                                {truth(item1?.ExpiresAt) ? (
                                  <>
                                    {text(
                                      formatDate(
                                        item1?.ExpiresAt,
                                        "02 Jan 2006",
                                      ),
                                    )}
                                  </>
                                ) : (
                                  <>No expiry</>
                                )}
                              </td>
                              <td>
                                <form method="POST" action="/trust">
                                  <input
                                    type="hidden"
                                    name="action"
                                    defaultValue="revoke"
                                  />
                                  <input
                                    type="hidden"
                                    name="badge_id"
                                    defaultValue={text(item1?.ID)}
                                  />
                                  <input
                                    type="hidden"
                                    name="owner_type"
                                    defaultValue={text(data?.OwnerType)}
                                  />
                                  <input
                                    type="hidden"
                                    name="owner_id"
                                    defaultValue={text(data?.OwnerID)}
                                  />
                                  <button type="submit">Remove badge</button>
                                </form>
                              </td>
                            </tr>
                          </Fragment>
                        ))
                      : null}
                  </tbody>
                </table>
              </div>
            </>
          ) : truth(data?.OwnerID > 0) ? (
            <>
              <p>No active trust badges for this account.</p>
            </>
          ) : null}
        </section>
      </main>
    </Workspace>
  );
}
