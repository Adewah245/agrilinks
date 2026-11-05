import Navbar from "../components/Navbar.jsx";
import FarmingLearningHub from "../components/FarmingLearningHub.jsx";
import VoiceControls from "../components/VoiceControls.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import { items, text, truth } from "../lib/view.js";
import mountLearningChat from "../behaviors/learning_chat.js";
import { usePageBehaviors } from "../lib/behaviors.js";

export default function LearningPage({ data }) {
  usePageBehaviors([mountLearningChat], data);
  return (
    <>
      <a className="learning-skip" href="#learning-main">
        Skip to learning
      </a>
      <Navbar className="navbar">
        <a className="logo" href="/" aria-label="Agrilinks Nigeria home">
          <img
            src="/static/assets/logo/agrilinks-nigeria-logo.svg"
            alt="Agrilinks Nigeria"
          />
        </a>
        <nav aria-label="Main navigation">
          <a href="/">Home</a>
          <a
            className="learning-nav-active"
            href="/learning"
            aria-current="page"
          >
            Learn with AI
          </a>
          <a href="/dashboard">Dashboard</a>
          <a href="/login">Login</a>
          <a href="/register">Register</a>
        </nav>
      </Navbar>
      <main id="learning-main" className="learning-page">
        <section className="learning-intro" aria-labelledby="learning-title">
          <p className="learning-eyebrow">Your farming questions welcome</p>
          <h1 id="learning-title">Learn farming with Agrilinks Nigeria AI</h1>
          <p>
            Ask about crops, livestock, poultry, fish farming, soil, or starting
            a farm. You can learn at your own pace, whether or not you are ready
            to buy or sell. No account is needed.
          </p>
        </section>
        <section className="learning-chat" aria-label="Farming learning chat">
          <div className="learning-chat-heading">
            <div>
              <h2>Ask a farming question</h2>
              <p>
                Agrilinks Nigeria AI will explain things in clear, practical
                steps.
              </p>
            </div>
            <button className="learning-reset" id="clear-chat" type="button">
              New chat
            </button>
          </div>
          {!truth(data?.AIAvailable) && (
            <p className="learning-availability" role="status">
              The learning guide is temporarily unavailable. Please try again
              later. You can still explore the farming guides below.
            </p>
          )}
          <div
            className="learning-messages"
            id="chat-messages"
            role="log"
            aria-live="polite"
            aria-relevant="additions"
          >
            <article className="learning-message learning-message-assistant">
              <span className="learning-message-label">
                Agrilinks Nigeria learning guide
              </span>
              <p>Hello! What would you like to learn about farming today?</p>
            </article>
          </div>
          <div className="learning-starters" aria-label="Example questions">
            <button
              type="button"
              data-category="Crop farming"
              data-question="How do I start growing vegetables in a small space?"
            >
              🌱 Growing vegetables
            </button>
            <button
              type="button"
              data-category="Poultry farming"
              data-question="What should I consider before starting poultry farming?"
            >
              🐓 Starting poultry
            </button>
            <button
              type="button"
              data-category="Crop farming"
              data-question="How can I improve the soil on my farm?"
            >
              🌿 Improving my soil
            </button>
          </div>
          <form id="learning-chat-form" className="learning-form">
            <label htmlFor="farming-type">
              Choose what you want to learn about
            </label>
            <Select id="farming-type" name="farming_type" required={true}>
              <option value="">Select a farming category</option>
              {items(data?.FarmingTypes).map((category) => (
                <option value={text(category)} key={text(category)}>
                  {text(category)}
                </option>
              ))}
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
              placeholder="Enter your farming category"
              hidden={true}
              disabled={true}
            />
            <label htmlFor="learning-question">Your question</label>
            <VoiceControls prefix="learning" />
            <div className="learning-compose">
              <Textarea
                id="learning-question"
                name="question"
                maxLength="1200"
                rows="2"
                placeholder="Ask anything about farming..."
                required={true}
              />
              <button
                id="send-question"
                type="submit"
                disabled={!truth(data?.AIAvailable)}
              >
                Ask AI
              </button>
            </div>
            <p className="learning-note" id="learning-status" role="status">
              This chat is for learning. For a serious crop or animal problem,
              ask a local farm adviser or animal doctor for help.
            </p>
          </form>
        </section>
        <div className="learning-signup">
          <FarmingLearningHub className="learning-guides-button">
            Explore farming topics
          </FarmingLearningHub>
          <p>
            Want to buy or sell later?{" "}
            <a href="/register">Create an Agrilinks Nigeria account</a> whenever
            you are ready.
          </p>
        </div>
      </main>
      <footer className="footer">
        <div className="footer-container">
          <div className="footer-brand">
            <img
              className="footer-brand-logo"
              src="/static/assets/logo/agrilinks-nigeria-logo.svg"
              alt="Agrilinks Nigeria"
            />
            <p>
              Empowering farmers through technology, learning, and digital
              agricultural commerce.
            </p>
          </div>
          <div className="footer-links">
            <h3>Quick Links</h3>
            <a href="/">Home</a>
            <a href="/learning">Learn with AI</a>
            <a href="/dashboard">Dashboard</a>
            <a href="/login">Login</a>
            <a href="/register">Register</a>
          </div>
        </div>
        <hr />
        <p className="copyright">
          © 2026 Agrilinks Nigeria. All rights reserved.
        </p>
      </footer>
    </>
  );
}
