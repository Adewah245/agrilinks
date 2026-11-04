import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, withValue, text, humanStatus } from "../lib/view.js";
export default function LearnPage({ data }) {
  return (
    <>
      <nav className="learning-nav" aria-label="Main navigation">
        <a href="/">Agrilinks Nigeria</a>
        <a href="/marketplace">Buy</a>
        <a href="/storage">Sell</a>
        <a href="/learn" aria-current="page">
          Learn
        </a>
        <a href="/chat">Messages</a>
        <a href="/profile">My profile</a>
        <a href="/wallet">Wallet</a>
      </nav>
      <main className="learning-main">
        <header>
          <p className="eyebrow">GROW YOUR KNOWLEDGE</p>
          <h1>Your learning space</h1>
          <p>
            Choose what you want to learn. Ask questions, plan your next steps,
            and keep track of what you have done.
          </p>
          <a href="/learning">Ask the farming guide a question →</a>
        </header>
        <div className="learning-columns">
          <section className="learning-card">
            <h2>Start something new</h2>
            {truth(data?.FarmingTypes) ? (
              <>
                <form method="post" action="/learn/start">
                  <label htmlFor="farming_type_id">
                    What would you like to learn?
                  </label>
                  <Select
                    id="farming_type_id"
                    name="farming_type_id"
                    required={true}
                  >
                    <option value="">Choose a farming activity</option>
                    {truth(data?.FarmingTypes)
                      ? items(data?.FarmingTypes).map((item1, index1) => (
                          <Fragment key={index1}>
                            <option value={text(item1?.ID)}>
                              {text(item1?.Name)}
                            </option>
                          </Fragment>
                        ))
                      : null}
                  </Select>
                  <label htmlFor="experience_level">
                    How much do you know already?
                  </label>
                  <Select id="experience_level" name="experience_level">
                    <option value="beginner">I am just starting</option>
                    <option value="some_experience">I know a little</option>
                    <option value="experienced">
                      I have farming experience
                    </option>
                  </Select>
                  <label htmlFor="goal">What would you like to achieve?</label>
                  <Textarea
                    id="goal"
                    name="goal"
                    maxLength="1000"
                    rows="3"
                  ></Textarea>
                  <button type="submit">Start learning</button>
                </form>
              </>
            ) : (
              <>
                <p>
                  We could not find any farming activities. Please try again
                  later.
                </p>
              </>
            )}
          </section>
          <section className="learning-card">
            <h2>Your learning journeys</h2>
            {truth(data?.Journeys) ? (
              items(data?.Journeys).map((item2, index2) => (
                <Fragment key={index2}>
                  <article className="journey-summary">
                    <h3>
                      {withValue(item2?.FarmingType, (item3) => (
                        <>{text(item3?.Name)}</>
                      ))}
                    </h3>
                    <p>
                      {text(humanStatus(item2?.Status))} · Your checklist is{" "}
                      {text(item2?.ReadinessScore)}% complete
                    </p>
                    <progress
                      value={text(item2?.ReadinessScore)}
                      max="100"
                      aria-label="Checklist progress"
                    >
                      {text(item2?.ReadinessScore)}%
                    </progress>
                    {truth(item2?.Goal) ? (
                      <>
                        <p>{text(item2?.Goal)}</p>
                      </>
                    ) : null}
                    <a href={"/learn/journey?id=" + text(item2?.ID)}>
                      Open this journey →
                    </a>
                  </article>
                </Fragment>
              ))
            ) : (
              <>
                <p>
                  Your learning journeys will appear here after you choose an
                  activity.
                </p>
              </>
            )}
          </section>
        </div>
        <section className="learning-card">
          <h2>Learn with other people</h2>
          <p>
            Ask questions and share what you are learning with other farmers and
            learners.
          </p>
          <a href="/learn/rooms">Find a learning room →</a>
        </section>
      </main>
    </>
  );
}
