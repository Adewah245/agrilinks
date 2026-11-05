import Workspace, {
  WorkspaceHeading,
  WorkspaceIcon,
} from "../components/Workspace.jsx";
import VoiceControls from "../components/VoiceControls.jsx";
import { Select, Textarea } from "../components/FormControls.jsx";
import { truth, text } from "../lib/view.js";
import mountAiAssistant from "../behaviors/ai_assistant.js";
import { usePageBehaviors } from "../lib/behaviors.js";

const suggestions = [
  [
    "Plant Problems",
    "My tomato leaves are turning yellow. What should I check?",
  ],
  ["Poultry", "My chickens are eating less than usual. What should I do?"],
  ["Storage", "How can I keep my maize dry and safe from insects?"],
];

export default function AiAssistantPage({ data }) {
  usePageBehaviors([mountAiAssistant], data);
  return (
    <Workspace current="/ai-assistant">
      <main className="ai-assistant">
        <WorkspaceHeading
          eyebrow="HELP FOR YOUR FARM"
          title="Your farming assistant"
          description="Tell us what you noticed. Get clear advice you can read or listen to."
        >
          <a className="assistant-history-link" href="/ai-diagnosis-history">
            Past advice →
          </a>
        </WorkspaceHeading>
        {truth(data?.Error) && (
          <div className="assistant-notice" role="alert">
            {text(data?.Error)}
          </div>
        )}
        {!truth(data?.Available) && (
          <div className="assistant-notice" role="status">
            <WorkspaceIcon name="assistant" />
            <div>
              <strong>The assistant cannot answer right now.</strong>
              <p>
                You can still explore our written farming guides while it is
                being connected.
              </p>
              <a href="/learning">Open farming guides →</a>
            </div>
          </div>
        )}
        <div className="assistant-layout">
          <section
            className="assistant-question-card"
            aria-labelledby="assistant-question-title"
          >
            <div className="assistant-card-heading">
              <span className="assistant-step">1</span>
              <div>
                <h2 id="assistant-question-title">Ask about your farm</h2>
                <p>Write a question, record your voice, or add a photo.</p>
              </div>
            </div>
            <form
              method="POST"
              action="/ai-assistant"
              encType="multipart/form-data"
              className="ai-form"
              id="ai-form"
            >
              <label htmlFor="category">What is your question about?</label>
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
                <option value="Feed and Nutrition">
                  {" "}
                  🌾 Feed and Nutrition{" "}
                </option>
                <option value="Storage"> 🏠 Agricultural Storage </option>
                <option value="Weather and Climate">
                  {" "}
                  🌦️ Weather and Climate{" "}
                </option>
                <option value="Farm Management"> 📋 Farm Management </option>
                <option value="Market and Pricing">
                  {" "}
                  💰 Market and Pricing{" "}
                </option>
                <option value="General Farming Questions">
                  {" "}
                  📝 General Farming Questions{" "}
                </option>
              </Select>
              <label id="composer-label" htmlFor="description">
                What happened, or what would you like to know?
              </label>
              <div
                className="assistant-composer"
                role="group"
                aria-labelledby="composer-label"
              >
                <Textarea
                  id="description"
                  name="description"
                  rows="6"
                  maxLength="5000"
                  placeholder="For example: My tomato leaves turned yellow three days ago. I water them every morning. What can I do?"
                  aria-describedby="description-help"
                />
                <input
                  type="file"
                  id="image"
                  name="image"
                  accept="image/*"
                  hidden
                />
                <input
                  type="file"
                  id="video"
                  name="video"
                  accept="video/*"
                  capture="environment"
                  hidden
                />
                <input
                  type="file"
                  id="audio"
                  name="audio"
                  accept="audio/*"
                  hidden
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
                    >
                      <span aria-hidden="true">＋</span> Add photo or video
                    </button>
                    <div
                      id="attachment-menu"
                      className="attachment-menu"
                      hidden
                    >
                      <button type="button" id="upload-photo">
                        Upload photo
                      </button>
                      <button type="button" id="take-photo">
                        Take photo
                      </button>
                      <button type="button" id="upload-video">
                        Upload video
                      </button>
                      <button type="button" id="record-video">
                        Record video
                      </button>
                    </div>
                  </div>
                  <button
                    type="button"
                    id="start-audio"
                    className="composer-icon microphone-button"
                    aria-label="Start voice recording"
                  >
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <path d="M12 15a4 4 0 0 0 4-4V6a4 4 0 1 0-8 0v5a4 4 0 0 0 4 4Zm-7-4a7 7 0 0 0 14 0M12 18v3m-3 0h6" />
                    </svg>{" "}
                    Record my voice
                  </button>
                  <button
                    type="button"
                    id="stop-audio"
                    className="stop-audio-button"
                    hidden
                    disabled
                  >
                    Finish recording
                  </button>
                </div>
              </div>
              <p id="description-help" className="media-help">
                Tell us what you grow or keep, what changed, and when it
                started. Photos and recordings are optional.
              </p>
              <VoiceControls prefix="diagnosis" showInput={false} />
              <div className="assistant-submit-row">
                <button
                  type="button"
                  id="think-toggle"
                  className="think-toggle"
                  aria-label="Use deeper AI analysis"
                  aria-pressed="false"
                >
                  <span aria-hidden="true">◉</span> Take a closer look
                </button>
                <button
                  type="submit"
                  className="send-button"
                  disabled={!truth(data?.Available)}
                >
                  Get farming advice <span aria-hidden="true">→</span>
                </button>
              </div>
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
                <div
                  id="video-feedback"
                  className="video-feedback"
                  hidden={true}
                >
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
          </section>
          <aside
            className="assistant-help-card"
            aria-labelledby="assistant-help-title"
          >
            <div className="assistant-help-icon">
              <WorkspaceIcon name="learning" />
            </div>
            <p className="workspace-eyebrow">A LITTLE HELP TO GET STARTED</p>
            <h2 id="assistant-help-title">One clear question at a time</h2>
            <p>
              Not sure what to ask? Choose an example and change it to fit your
              farm.
            </p>
            <div className="assistant-suggestions">
              {suggestions.map(([category, question]) => (
                <button
                  key={category}
                  type="button"
                  data-assistant-category={category}
                  data-assistant-question={question}
                >
                  {question}
                  <span aria-hidden="true">↗</span>
                </button>
              ))}
            </div>
            <div className="assistant-help-tip">
              <strong>Show us what you see</strong>
              <p>
                A clear photo of the affected leaf, animal, or stored food can
                help the assistant understand your question.
              </p>
            </div>
            <a href="/learning">Explore farming guides →</a>
          </aside>
        </div>
        <div className="assistant-answer-card">
          <div
            id="ai-result-empty"
            className="assistant-answer-empty"
            hidden={truth(data?.Result)}
          >
            <span className="assistant-step">2</span>
            <div>
              <h2>Your advice will appear here</h2>
              <p>
                Read the answer, listen to it, or ask for an example picture
                when one is available.
              </p>
            </div>
          </div>
          <section
            id="ai-result"
            className="ai-result"
            aria-live="polite"
            aria-label="Farming advice"
            hidden={!truth(data?.Result)}
          >
            <h2>Your farming advice</h2>
            <div id="ai-result-text" className="ai-result-text">
              {text(data?.Result)}
            </div>
            <div id="ai-result-tools"></div>
          </section>
        </div>
      </main>
    </Workspace>
  );
}
