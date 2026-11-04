import Navbar from "../components/Navbar.jsx";
import { Select } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text, formatDate } from "../lib/view.js";
import mountAiDiagnosisHistory from "../behaviors/ai-diagnosis-history.js";
import { usePageBehaviors } from "../lib/behaviors.js";
export default function AiDiagnosisHistoryPage({ data }) {
  usePageBehaviors([mountAiDiagnosisHistory], data);
  return (
    <>
      <Navbar className="navbar">
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
        <nav aria-label="Main navigation">
          <a href="/dashboard"> Dashboard </a>
          <a href="/storage"> Storage </a>
          <a href="/marketplace"> Marketplace </a>
          <a href="/negotiations"> Negotiations </a>
          <a href="/cart"> Cart </a>
          <a href="/chat"> Chat </a>
          <a href="/ai-assistant" aria-current="page">
            {" "}
            AI Assistant{" "}
          </a>
          <a href="/notifications"> Notifications </a>
          <a href="/profile"> Profile </a>
          <a href="/logout"> Logout </a>
        </nav>
      </Navbar>
      <main className="diagnosis-history">
        <section className="page-header">
          <div>
            <a href="/ai-assistant" className="back-link">
              {" "}
              ← AI Assistant{" "}
            </a>
            <h1> AI Diagnosis History </h1>
            <p>
              {" "}
              Review your previous farming consultations and AI-powered
              diagnoses.{" "}
            </p>
          </div>
          <a href="/ai-assistant" className="new-diagnosis-button">
            {" "}
            🤖 New Diagnosis{" "}
          </a>
        </section>
        <section
          className="history-controls"
          aria-label="Diagnosis history filters"
        >
          <div className="search-wrapper">
            <label htmlFor="history-search" className="visually-hidden">
              {" "}
              Search diagnosis history{" "}
            </label>
            <span className="search-icon" aria-hidden="true">
              {" "}
              🔍{" "}
            </span>
            <input
              type="search"
              id="history-search"
              placeholder="Search diagnosis history..."
              autoComplete="off"
              aria-label="Search diagnosis history"
            />
          </div>
          <div className="filter-wrapper">
            <label htmlFor="category-filter"> Category </label>
            <Select
              id="category-filter"
              aria-label="Filter diagnoses by farming category"
            >
              <option value="all"> All Categories </option>
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
          </div>
        </section>
        <div className="history-summary" aria-live="polite">
          <span> Your AI consultations </span>
          <strong id="result-count"> 0 </strong>
        </div>
        <section
          className="history-container"
          aria-label="AI diagnosis history"
        >
          {truth(data?.History) ? (
            <>
              <div className="diagnosis-list" id="diagnosis-list">
                {truth(data?.History)
                  ? items(data?.History).map((item1, index1) => (
                      <Fragment key={index1}>
                        <article
                          className="diagnosis-card"
                          data-category={text(item1?.CategoryText)}
                          data-search={
                            text(item1?.CategoryText) +
                            " " +
                            text(item1?.SymptomsText) +
                            " " +
                            text(item1?.ResultText)
                          }
                        >
                          <div className="diagnosis-card-header">
                            <div className="diagnosis-category">
                              <span
                                className="category-icon"
                                aria-hidden="true"
                              >
                                {truth(item1?.CategoryText === "Crops") ? (
                                  <> 🌾 </>
                                ) : truth(
                                    item1?.CategoryText === "Livestock",
                                  ) ? (
                                  <> 🐄 </>
                                ) : truth(item1?.CategoryText === "Poultry") ? (
                                  <> 🐔 </>
                                ) : truth(item1?.CategoryText === "Goats") ? (
                                  <> 🐐 </>
                                ) : truth(item1?.CategoryText === "Sheep") ? (
                                  <> 🐑 </>
                                ) : truth(item1?.CategoryText === "Cattle") ? (
                                  <> 🐄 </>
                                ) : truth(item1?.CategoryText === "Pigs") ? (
                                  <> 🐖 </>
                                ) : truth(item1?.CategoryText === "Fish") ? (
                                  <> 🐟 </>
                                ) : truth(
                                    item1?.CategoryText === "Beekeeping",
                                  ) ? (
                                  <> 🐝 </>
                                ) : truth(
                                    item1?.CategoryText === "Pests/Insects",
                                  ) ? (
                                  <> 🐛 </>
                                ) : truth(
                                    item1?.CategoryText === "Plant Problems",
                                  ) ? (
                                  <> 🌱 </>
                                ) : truth(
                                    item1?.CategoryText === "Animal Health",
                                  ) ? (
                                  <> 🩺 </>
                                ) : truth(
                                    item1?.CategoryText === "Fish Health",
                                  ) ? (
                                  <> 🐟 </>
                                ) : truth(
                                    item1?.CategoryText ===
                                      "Feed and Nutrition",
                                  ) ? (
                                  <> 🌾 </>
                                ) : truth(item1?.CategoryText === "Storage") ? (
                                  <> 🏠 </>
                                ) : truth(
                                    item1?.CategoryText ===
                                      "Weather and Climate",
                                  ) ? (
                                  <> 🌦️ </>
                                ) : truth(
                                    item1?.CategoryText === "Farm Management",
                                  ) ? (
                                  <> 📋 </>
                                ) : truth(
                                    item1?.CategoryText ===
                                      "Market and Pricing",
                                  ) ? (
                                  <> 💰 </>
                                ) : (
                                  <> 📝 </>
                                )}
                              </span>
                              <div>
                                <h2>{text(item1?.CategoryText)}</h2>
                                <time
                                  dateTime={text(
                                    formatDate(
                                      item1?.CreatedAt,
                                      "2006-01-02T15:04:05Z07:00",
                                    ),
                                  )}
                                >
                                  {text(
                                    formatDate(
                                      item1?.CreatedAt,
                                      "02 Jan 2006, 03:04 PM",
                                    ),
                                  )}
                                </time>
                              </div>
                            </div>
                            <span
                              className="diagnosis-id"
                              aria-label={"Diagnosis ID " + text(item1?.ID)}
                            >
                              {" "}
                              #{text(item1?.ID)}
                            </span>
                          </div>
                          <div className="diagnosis-description">
                            <h3> Problem Description </h3>
                            {truth(item1?.SymptomsText) ? (
                              <>
                                <p>{text(item1?.SymptomsText)}</p>
                              </>
                            ) : (
                              <>
                                <p className="empty-description">
                                  {" "}
                                  No symptoms were provided.{" "}
                                </p>
                              </>
                            )}
                          </div>
                          <div className="diagnosis-result">
                            <div className="result-header">
                              <h3> 🤖 AI Diagnosis </h3>
                            </div>
                            <div className="result-content">
                              {text(item1?.ResultText)}
                            </div>
                          </div>
                          <div className="diagnosis-card-footer">
                            <time
                              className="diagnosis-date"
                              dateTime={text(
                                formatDate(item1?.CreatedAt, "2006-01-02"),
                              )}
                            >
                              {text(
                                formatDate(item1?.CreatedAt, "02 Jan 2006"),
                              )}
                            </time>
                            <button
                              type="button"
                              className="view-result-button"
                              aria-label={
                                "View full result for diagnosis #" +
                                text(item1?.ID)
                              }
                            >
                              {" "}
                              View Full Result{" "}
                            </button>
                            <form
                              method="POST"
                              action="/ai-diagnosis-history"
                              className="delete-diagnosis-form"
                            >
                              <input
                                type="hidden"
                                name="id"
                                defaultValue={text(item1?.ID)}
                              />
                              <button
                                type="submit"
                                className="delete-diagnosis-button"
                                aria-label={
                                  "Delete diagnosis #" + text(item1?.ID)
                                }
                              >
                                {" "}
                                Delete{" "}
                              </button>
                            </form>
                          </div>
                        </article>
                      </Fragment>
                    ))
                  : null}
              </div>
              <div
                id="no-search-results"
                className="no-results"
                hidden={true}
                role="status"
              >
                <div className="no-results-icon" aria-hidden="true">
                  {" "}
                  🔍{" "}
                </div>
                <h2> No diagnoses found </h2>
                <p> Try changing your search term or category filter. </p>
              </div>
            </>
          ) : (
            <>
              <div className="empty-history">
                <div className="empty-history-icon" aria-hidden="true">
                  {" "}
                  🤖{" "}
                </div>
                <h2> No AI diagnoses yet </h2>
                <p>
                  {" "}
                  You haven't used Agrilinks Nigeria AI for a diagnosis yet.
                  Start your first consultation to get agricultural
                  assistance.{" "}
                </p>
                <a href="/ai-assistant" className="new-diagnosis-button">
                  {" "}
                  Start Your First Diagnosis{" "}
                </a>
              </div>
            </>
          )}
        </section>
      </main>
      <div
        className="result-modal"
        id="result-modal"
        hidden={true}
        aria-hidden="true"
      >
        <div
          className="modal-overlay"
          id="modal-overlay"
          aria-hidden="true"
        ></div>
        <div
          className="modal-content"
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-title"
          aria-describedby="modal-date"
        >
          <button
            type="button"
            className="modal-close"
            id="modal-close"
            aria-label="Close diagnosis result"
            title="Close"
          >
            {" "}
            ×{" "}
          </button>
          <div className="modal-header">
            <span className="modal-icon" aria-hidden="true">
              {" "}
              🤖{" "}
            </span>
            <div>
              <h2 id="modal-title"> AI Diagnosis </h2>
              <p id="modal-date"></p>
            </div>
          </div>
          <div className="modal-body">
            <div className="modal-section">
              <h3> Category </h3>
              <p id="modal-category"></p>
            </div>
            <div className="modal-section">
              <h3> Problem Description </h3>
              <p id="modal-description"></p>
            </div>
            <div className="modal-section">
              <h3> AI Diagnosis </h3>
              <div className="modal-result" id="modal-result"></div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
