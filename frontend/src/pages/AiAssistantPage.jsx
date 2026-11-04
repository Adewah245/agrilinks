import Navbar from "../components/Navbar.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import "react";
import { truth, text } from "../lib/view.js";
import mountAiAssistant from "../behaviors/ai_assistant.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function AiAssistantPage({ data }) {
  usePageBehaviors([mountAiAssistant], data);
  return (
    <>
      <Navbar className="navbar">
        <div className="navbar-container">
          <a
            className="logo"
            href="/dashboard"
            aria-label="Agrilinks Nigeria Dashboard"
          >
            <img
              src="/static/assets/logo/agrilinks-nigeria-logo.svg"
              alt="Agrilinks Nigeria"
            />
          </a>
          <button
            className="nav-toggle"
            id="navToggle"
            aria-label="Toggle navigation menu"
            aria-expanded="false"
            aria-controls="navLinks"
          >
            <span className="hamburger"></span>
          </button>
          <nav className="nav-links" id="navLinks" aria-label="Main navigation">
            <a href="/dashboard">Dashboard</a>
            <a href="/storage">Storage</a>
            <a href="/marketplace">Marketplace</a>
            <a href="/negotiations">Negotiations</a>
            <a href="/cart">Cart</a>
            <a href="/chat">Chat</a>
            <a href="/ai-assistant" aria-current="page">
              {" "}
              AI Assistant{" "}
            </a>
            <a href="/profile">Profile</a>
            <a href="/notifications">Notifications</a>
            <a href="/logout" className="nav-logout">
              {" "}
              Logout{" "}
            </a>
          </nav>
        </div>
      </Navbar>
      <main className="ai-assistant">
        <h1>🤖 Agrilinks Nigeria AI Assistant</h1>
        <p className="assistant-intro">
          {" "}
          Ask questions about farming, describe a problem, or provide photo,
          video, or voice context for analysis.{" "}
        </p>
        {truth(data?.Error) ? (
          <>
            <div className="message error-message" role="alert">
              {text(data?.Error)}
            </div>
          </>
        ) : null}
        <form
          method="POST"
          action="/ai-assistant"
          encType="multipart/form-data"
          className="ai-form"
          id="ai-form"
        >
          <label htmlFor="category"> Farming Category </label>
          <Select id="category" name="category" required={true}>
            <option value=""> Select a category </option>
            <option value="Crops"> 🌾 Crops </option>
            <option value="Livestock"> 🐄 Livestock </option>
            <option value="Poultry"> 🐔 Poultry </option>
            <option value="Goats"> 🐐 Goats </option>
            <option value="Sheep"> 🐑 Sheep </option>
            <option value="Cattle"> 🐄 Cattle </option>
            <option value="Pigs"> 🐖 Pigs </option>
            <option value="Fish"> 🐟 Fish / Aquaculture </option>
            <option value="Beekeeping"> 🐝 Beekeeping </option>
            <option value="Pests/Insects"> 🐛 Pests / Insects </option>
            <option value="Plant Problems"> 🌱 Plant Problems </option>
            <option value="Animal Health"> 🩺 Animal Health </option>
            <option value="Fish Health"> 🐟 Fish Health </option>
            <option value="Feed and Nutrition"> 🌾 Feed and Nutrition </option>
            <option value="Storage"> 🏠 Agricultural Storage </option>
            <option value="Weather and Climate">
              {" "}
              🌦️ Weather and Climate{" "}
            </option>
            <option value="Farm Management"> 📋 Farm Management </option>
            <option value="Market and Pricing"> 💰 Market and Pricing </option>
            <option value="General Farming Questions">
              {" "}
              📝 General Farming Questions{" "}
            </option>
          </Select>
          <label id="composer-label" htmlFor="description">
            {" "}
            What happened?{" "}
          </label>
          <div
            className="assistant-composer"
            role="group"
            aria-labelledby="composer-label"
          >
            <Textarea
              id="description"
              name="description"
              rows="5"
              maxLength="5000"
              placeholder="Describe what you noticed, what you are trying to do, or the question you want to ask..."
              aria-describedby="description-help"
            ></Textarea>
            <input
              type="file"
              id="image"
              name="image"
              accept="image/*"
              hidden={true}
            />
            <input
              type="file"
              id="video"
              name="video"
              accept="video/*"
              capture="environment"
              hidden={true}
            />
            <input
              type="file"
              id="audio"
              name="audio"
              accept="audio/*"
              hidden={true}
            />
            <input
              type="hidden"
              id="thinking"
              name="thinking"
              defaultValue="false"
            />
            <div className="composer-controls">
              <div className="attachment-menu-wrap">
                <button
                  type="button"
                  id="add-image"
                  className="composer-icon plus-button"
                  aria-label="Add media"
                  aria-expanded="false"
                  aria-controls="attachment-menu"
                  title="Add media"
                >
                  {" "}
                  +{" "}
                </button>
                <div
                  id="attachment-menu"
                  className="attachment-menu"
                  hidden={true}
                >
                  <button type="button" id="upload-photo">
                    {" "}
                    📷 Upload photo{" "}
                  </button>
                  <button type="button" id="take-photo">
                    {" "}
                    📸 Take photo{" "}
                  </button>
                  <button type="button" id="upload-video">
                    {" "}
                    🎥 Upload video{" "}
                  </button>
                  <button type="button" id="record-video">
                    {" "}
                    🎬 Record video{" "}
                  </button>
                </div>
              </div>
              <button
                type="button"
                id="think-toggle"
                className="think-toggle"
                aria-label="Use deeper AI analysis"
                aria-pressed="false"
                title="Use deeper AI analysis"
              >
                <span aria-hidden="true">◉</span>
                <span>Think</span>
              </button>
              <button
                type="button"
                id="start-audio"
                className="composer-icon microphone-button"
                aria-label="Start voice recording"
                title="Record voice"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                  <path d="M12 15a4 4 0 0 0 4-4V6a4 4 0 1 0-8 0v5a4 4 0 0 0 4 4Zm-7-4a7 7 0 0 0 14 0M12 18v3m-3 0h6" />
                </svg>
              </button>
              <button
                type="button"
                id="stop-audio"
                className="stop-audio-button"
                hidden={true}
                disabled={true}
              >
                {" "}
                Stop recording{" "}
              </button>
              <button
                type="submit"
                className="send-button"
                aria-label="Send to AI"
                title="Send to AI"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                  <path d="m4 12 16-8-6 16-3-7-7-1Z" />
                  <path d="m11 13 4-4" />
                </svg>
              </button>
            </div>
          </div>
          <p id="description-help" className="media-help">
            {" "}
            You can describe the issue or attach a photo, video, or voice
            recording. Media uploads are optional.{" "}
          </p>
          <div className="composer-feedback">
            <div
              id="image-preview-wrap"
              className="attachment-preview"
              hidden={true}
            >
              <img
                id="image-preview"
                alt="Selected agricultural image preview"
              />
              <div className="attachment-info">
                <strong> Photo attached </strong>
                <span id="image-status" aria-live="polite"></span>
              </div>
              <button
                type="button"
                id="remove-image"
                className="remove-attachment"
              >
                {" "}
                Remove image{" "}
              </button>
            </div>
            <div id="video-feedback" className="video-feedback" hidden={true}>
              <div className="video-feedback-content">
                <strong> 🎥 Video attached </strong>
                <span id="video-status" aria-live="polite"></span>
              </div>
              <video
                id="video-preview"
                controls={true}
                playsInline={true}
                hidden={true}
                aria-label="Selected video preview"
              ></video>
              <button
                type="button"
                id="remove-video"
                className="remove-attachment"
              >
                {" "}
                Remove video{" "}
              </button>
            </div>
            <p
              id="audio-status"
              className="recording-status"
              aria-live="polite"
            ></p>
            <audio
              id="audio-preview"
              controls={true}
              hidden={true}
              aria-label="Voice recording preview"
            ></audio>
            <button
              type="button"
              id="play-audio"
              className="remove-attachment"
              hidden={true}
            >
              {" "}
              Play recording{" "}
            </button>
            <button
              type="button"
              id="remove-audio"
              className="remove-attachment"
              hidden={true}
            >
              {" "}
              Remove voice recording{" "}
            </button>
          </div>
          <div
            id="submit-status"
            className="submit-status"
            role="status"
            aria-live="polite"
          ></div>
        </form>
        {truth(data?.Result) ? (
          <>
            <section
              className="ai-result"
              aria-live="polite"
              aria-label="AI diagnosis result"
            >
              <h2> 🤖 AI Result </h2>
              <div className="ai-result-text">{text(data?.Result)}</div>
            </section>
          </>
        ) : null}
        <p className="history-link-wrap">
          <a className="history-link" href="/ai-diagnosis-history">
            {" "}
            View AI diagnosis history{" "}
          </a>
        </p>
      </main>
    </>
  );
}
