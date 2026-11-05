import Workspace from "../components/Workspace.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, withValue, text, humanStatus } from "../lib/view.js";
import mountLearning from "../behaviors/learning.js";
import { usePageBehaviors } from "../lib/behaviors.js";
import VoiceControls from "../components/VoiceControls.jsx";
export default function LearningJourneyPage({ data }) {
  usePageBehaviors([mountLearning], data);
  return (
    <Workspace current="/learn">
      <main className="learning-main">
        <header>
          <p className="eyebrow">ONE STEP AT A TIME</p>
          <h1>
            {withValue(
              data?.Journey?.FarmingType,
              (item1) => (
                <>{text(item1?.Name)}</>
              ),
              () => (
                <>Your learning journey</>
              ),
            )}
          </h1>
          {truth(data?.Journey?.Goal) ? (
            <>
              <p>{text(data?.Journey?.Goal)}</p>
            </>
          ) : null}
          <p>
            {text(humanStatus(data?.Journey?.Status))} · Your checklist is{" "}
            {text(data?.Journey?.ReadinessScore)}% complete
          </p>
          <progress
            value={text(data?.Journey?.ReadinessScore)}
            max="100"
            aria-label="Checklist progress"
          >
            {text(data?.Journey?.ReadinessScore)}%
          </progress>
        </header>
        <p
          id="learning-error"
          className="learning-error"
          role="alert"
          hidden={true}
        ></p>
        <div className="learning-columns learning-workspace">
          <section className="learning-card">
            <h2>Your farming guide</h2>
            <p>
              Ask about getting started, caring for your farm, or your next
              step.
            </p>
            <div className="learning-messages" aria-live="polite">
              {truth(data?.Messages) ? (
                items(data?.Messages).map((item2, index2) => (
                  <Fragment key={index2}>
                    <article
                      className={"learning-message " + text(item2?.SenderType)}
                    >
                      <strong>
                        {truth(item2?.SenderType === "ai") ? (
                          <>Agrilinks Nigeria guide</>
                        ) : truth(item2?.SenderType === "user") ? (
                          <>You</>
                        ) : (
                          <>Learning note</>
                        )}
                      </strong>
                      <p>{text(item2?.Content)}</p>
                    </article>
                  </Fragment>
                ))
              ) : (
                <>
                  <p>Start with a question about what you want to learn.</p>
                </>
              )}
            </div>
            {truth(data?.Journey?.Status === "active") ? (
              <>
                {truth(data?.AIAvailable) ? (
                  <>
                    <form
                      method="post"
                      action="/learn/message"
                      data-learning-form=""
                    >
                      <input
                        type="hidden"
                        name="journey_id"
                        defaultValue={text(data?.Journey?.ID)}
                      />
                      <label htmlFor="learning-question">Your question</label>
                      <VoiceControls prefix="journey" />
                      <Textarea
                        id="learning-question"
                        name="content"
                        rows="3"
                        maxLength="10000"
                        required={true}
                      ></Textarea>
                      <button type="submit">Ask your guide</button>
                      <p className="form-status" role="status"></p>
                    </form>
                  </>
                ) : (
                  <>
                    <p>
                      The guide is unavailable at the moment. You can still work
                      on your checklist.
                    </p>
                  </>
                )}
              </>
            ) : (
              <>
                <p>Resume this journey to ask your guide another question.</p>
              </>
            )}
          </section>
          <section className="learning-card">
            <h2>Your next steps</h2>
            <p>
              Add steps from what you learn. Update each one as you make
              progress.
            </p>
            {truth(data?.Checkpoints) ? (
              items(data?.Checkpoints).map((item3, index3) => (
                <Fragment key={index3}>
                  <article className="learning-checkpoint">
                    <h3>{text(item3?.Title)}</h3>
                    {truth(item3?.Description) ? (
                      <>
                        <p>{text(item3?.Description)}</p>
                      </>
                    ) : null}
                    <p>
                      {text(humanStatus(item3?.Status))} ·{" "}
                      {text(item3?.Progress)}%
                    </p>
                    {truth(data?.Journey?.Status === "active") ? (
                      <>
                        <form
                          method="post"
                          action="/learn/checkpoint/update"
                          data-learning-form=""
                        >
                          <input
                            type="hidden"
                            name="journey_id"
                            defaultValue={text(data?.Journey?.ID)}
                          />
                          <input
                            type="hidden"
                            name="checkpoint_id"
                            defaultValue={text(item3?.ID)}
                          />
                          <label htmlFor={"step-status-" + text(item3?.ID)}>
                            How is it going?
                          </label>
                          <Select
                            id={"step-status-" + text(item3?.ID)}
                            name="status"
                          >
                            <option
                              value="pending"
                              selected={truth(item3?.Status === "pending")}
                            >
                              Not started
                            </option>
                            <option
                              value="in_progress"
                              selected={truth(item3?.Status === "in_progress")}
                            >
                              Working on it
                            </option>
                            <option
                              value="completed"
                              selected={truth(item3?.Status === "completed")}
                            >
                              Done
                            </option>
                            <option
                              value="not_applicable"
                              selected={truth(
                                item3?.Status === "not_applicable",
                              )}
                            >
                              Not needed for my plan
                            </option>
                          </Select>
                          <label htmlFor={"step-progress-" + text(item3?.ID)}>
                            How much have you done? (%)
                          </label>
                          <input
                            id={"step-progress-" + text(item3?.ID)}
                            type="number"
                            name="progress"
                            min="0"
                            max="100"
                            defaultValue={text(item3?.Progress)}
                            required={true}
                          />
                          <label htmlFor={"step-notes-" + text(item3?.ID)}>
                            Your notes
                          </label>
                          <Textarea
                            id={"step-notes-" + text(item3?.ID)}
                            name="notes"
                            maxLength="2000"
                            rows="2"
                          >
                            {text(item3?.Notes)}
                          </Textarea>
                          <button type="submit">Save this step</button>
                        </form>
                      </>
                    ) : null}
                  </article>
                </Fragment>
              ))
            ) : (
              <>
                <p>You have not added any steps yet.</p>
              </>
            )}
            {truth(data?.Journey?.Status === "active") ? (
              <>
                <details className="add-step">
                  <summary>Add a step</summary>
                  <form
                    method="post"
                    action="/learn/checkpoint/create"
                    data-learning-form=""
                  >
                    <input
                      type="hidden"
                      name="journey_id"
                      defaultValue={text(data?.Journey?.ID)}
                    />
                    <label htmlFor="step-title">What will you do?</label>
                    <input
                      id="step-title"
                      name="title"
                      maxLength="255"
                      required={true}
                    />
                    <label htmlFor="step-description">
                      Any details to remember?
                    </label>
                    <Textarea
                      id="step-description"
                      name="description"
                      maxLength="2000"
                      rows="2"
                    ></Textarea>
                    <button type="submit">Add to my checklist</button>
                  </form>
                </details>
              </>
            ) : null}
          </section>
        </div>
        <section className="learning-card">
          <h2>Manage this journey</h2>
          <p>
            This checklist records your own progress. It does not certify your
            farm or replace advice from a qualified local expert.
          </p>
          <form
            className="journey-actions"
            method="post"
            action="/learn/journey/update"
          >
            <input
              type="hidden"
              name="journey_id"
              defaultValue={text(data?.Journey?.ID)}
            />
            {truth(data?.Journey?.Status === "active") ? (
              <>
                <button name="status" value="paused">
                  Pause learning
                </button>
                <button name="status" value="completed">
                  I have finished my checklist
                </button>
              </>
            ) : (
              <>
                <button name="status" value="active">
                  Resume learning
                </button>
              </>
            )}
          </form>
        </section>
      </main>
    </Workspace>
  );
}
