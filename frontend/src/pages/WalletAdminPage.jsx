import Workspace, { WorkspaceHeading } from "../components/Workspace.jsx";
import { Fragment } from "react";
import { truth, items, text, humanStatus } from "../lib/view.js";
export default function WalletAdminPage({ data }) {
  return (
    <Workspace current="/wallet/admin" staff>
      <main className="learning-page account-page">
        <WorkspaceHeading
          eyebrow="STAFF WORKSPACE"
          title="Wallet requests"
          description="Review money requests and follow each payment."
        />
        {truth(data?.Error) ? (
          <>
            <p role="alert">{text(data?.Error)}</p>
          </>
        ) : null}
        <section className="panel">
          <h2>Withdrawals</h2>
          {truth(data?.Withdrawals) ? (
            items(data?.Withdrawals).map((item1, index1) => (
              <Fragment key={index1}>
                <article>
                  <p>
                    {text(item1?.FullName)} · {text(item1?.Currency)}
                    {text(item1?.Amount)} · {text(humanStatus(item1?.Status))}
                  </p>
                  <p>
                    {text(item1?.BankName)} · {text(item1?.AccountName)} ·{" "}
                    {text(item1?.MaskedAccountNumber)}
                  </p>
                  <p>Reference: {text(item1?.Reference)}</p>
                  <form method="POST" action="/wallet/admin">
                    <input
                      type="hidden"
                      name="kind"
                      defaultValue="withdrawal"
                    />
                    <input
                      type="hidden"
                      name="id"
                      defaultValue={text(item1?.ID)}
                    />
                    {truth(item1?.Status === "pending") ? (
                      <>
                        <button name="decision" value="approve">
                          Check bank and approve
                        </button>
                        <button name="decision" value="rejected">
                          Reject and return money
                        </button>
                      </>
                    ) : truth(item1?.Status === "processing") ? (
                      <>
                        <button name="decision" value="check">
                          Check payment status
                        </button>
                      </>
                    ) : null}
                  </form>
                </article>
              </Fragment>
            ))
          ) : (
            <>
              <p>No withdrawal requests.</p>
            </>
          )}
        </section>
        <section className="panel">
          <h2>Wallet transfers</h2>
          {truth(data?.Transfers) ? (
            items(data?.Transfers).map((item2, index2) => (
              <Fragment key={index2}>
                <article>
                  <p>
                    {text(item2?.SenderName)} → {text(item2?.RecipientName)} ·{" "}
                    {text(item2?.Currency)}
                    {text(item2?.Amount)} · {text(humanStatus(item2?.Status))}
                  </p>
                  <p>{text(item2?.Note)}</p>
                  {truth(item2?.Status === "pending") ? (
                    <>
                      <form method="POST" action="/wallet/admin">
                        <input
                          type="hidden"
                          name="kind"
                          defaultValue="transfer"
                        />
                        <input
                          type="hidden"
                          name="id"
                          defaultValue={text(item2?.ID)}
                        />
                        <button name="decision" value="completed">
                          Approve transfer
                        </button>
                        <button name="decision" value="rejected">
                          Reject and return money
                        </button>
                      </form>
                    </>
                  ) : null}
                </article>
              </Fragment>
            ))
          ) : (
            <>
              <p>No transfers.</p>
            </>
          )}
        </section>
        <section className="panel">
          <h2>Payments needing a check</h2>
          <p>
            These payments arrived after their order had already been paid or
            closed. Check them in Flutterwave and arrange any refund there.
          </p>
          {truth(data?.Payments) ? (
            items(data?.Payments).map((item3, index3) => (
              <Fragment key={index3}>
                <p>
                  Order #{text(item3?.OrderID)} · {text(item3?.Currency)}
                  {text(item3?.Amount)} · {text(item3?.Reference)}
                </p>
              </Fragment>
            ))
          ) : (
            <>
              <p>No extra payments need a check.</p>
            </>
          )}
        </section>
      </main>
    </Workspace>
  );
}
