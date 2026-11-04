import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text } from "../lib/view.js";
import mountLearningChat from "../behaviors/learning_chat.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function LearningPage({ data }) {
  usePageBehaviors([mountLearningChat], data);
  return (
    <>
      <a className="learning-skip" href="#learning-main">
        Skip to learning
      </a>
      <header className="public-learning-nav">
        <a className="learning-brand" href="/">
          Agrilinks Nigeria<span>Learn. Grow. Connect.</span>
        </a>
        <nav aria-label="Main navigation">
          <a href="/">Home</a>
          <a href="/learning" aria-current="page">
            Learn with AI
          </a>
          <a href="/learn">My journeys</a>
          <a href="/login">Sign in</a>
        </nav>
      </header>
      <main id="learning-main" className="public-learning-main">
        <header className="public-learning-intro">
          <p className="learning-eyebrow">YOUR FARMING QUESTIONS, WELCOME</p>
          <h1>Learn farming, one question at a time.</h1>
          <p>
            Practical guidance for your next step. Explore crops, livestock and
            starting a farm at your own pace. You can begin without an account.
          </p>
        </header>
        <section
          className="public-learning-chat"
          aria-labelledby="learning-chat-title"
        >
          <div className="learning-chat-heading">
            <div>
              <h2 id="learning-chat-title">Your farming guide</h2>
              <p>Clear explanations. Practical steps.</p>
            </div>
            <button id="clear-chat" className="learning-reset" type="button">
              New chat
            </button>
          </div>
          {truth(!truth(data?.AIAvailable)) ? (
            <>
              <p className="learning-availability" role="status">
                The learning guide is temporarily unavailable. Please try again
                later. You can still explore the categories below.
              </p>
            </>
          ) : null}
          <div
            id="chat-messages"
            className="public-learning-messages"
            role="log"
            aria-live="polite"
            aria-relevant="additions"
          >
            <article className="public-learning-message assistant">
              <span>Agrilinks Nigeria guide</span>
              <p>Hello! What would you like to learn about farming today?</p>
            </article>
          </div>
          <div className="learning-starters" aria-label="Example questions">
            <button
              type="button"
              data-category="Crop farming"
              data-question="How do I start growing vegetables in a small space?"
            >
              Growing vegetables
            </button>
            <button
              type="button"
              data-category="Poultry farming"
              data-question="What should I consider before starting poultry farming?"
            >
              Starting poultry
            </button>
            <button
              type="button"
              data-category="Soil management"
              data-question="How can I improve the soil on my farm?"
            >
              Improving my soil
            </button>
          </div>
          <form id="learning-chat-form" className="public-learning-form">
            <label htmlFor="farming-type">
              What would you like to learn about?
            </label>
            <Select id="farming-type" required={true}>
              <option value="">Choose a farming category</option>
              {truth(data?.FarmingTypes)
                ? items(data?.FarmingTypes).map((item1, index1) => (
                    <Fragment key={index1}>
                      <option value={text(item1)}>{text(item1)}</option>
                    </Fragment>
                  ))
                : null}
              <option value="other">Other</option>
            </Select>
            <label
              htmlFor="custom-farming-type"
              id="custom-farming-label"
              hidden={true}
            >
              Your farming category
            </label>
            <input
              id="custom-farming-type"
              maxLength="80"
              placeholder="For example, beekeeping"
              hidden={true}
              disabled={true}
            />
            <label htmlFor="learning-question">Your question</label>
            <div className="learning-compose">
              <Textarea
                id="learning-question"
                maxLength="1200"
                rows="3"
                placeholder="Ask a question about farming…"
                required={true}
              ></Textarea>
              <button
                id="send-question"
                type="submit"
                disabled={truth(!truth(data?.AIAvailable))}
              >
                Ask your guide
              </button>
            </div>
            <p
              id="learning-status"
              className="public-learning-status"
              role="status"
            >
              Choose a category to begin. Your conversation stays in this tab.
            </p>
          </form>
        </section>
        <div className="learning-next">
          <div>
            <h2>Build on what you learn.</h2>
            <p>
              Save a learning journey, make a checklist and learn with other
              people.
            </p>
          </div>
          <a href="/learn">Open my learning space →</a>
        </div>
        <p className="learning-footnote">
          For urgent crop or animal health problems, contact a local extension
          worker or veterinarian.
        </p>
      </main>
      <footer className="public-learning-footer">
        <a href="/">Agrilinks Nigeria</a>
        <span>Growing knowledge. Growing possibilities.</span>
        <a href="/register">Create an account</a>
      </footer>
    </>
  );
}
