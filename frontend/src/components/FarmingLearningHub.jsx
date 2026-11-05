import { useEffect, useRef, useState } from "react";
import { farmingGuides } from "../lib/farming-guides.js";

const learningURL = (topic) =>
  topic ? `/learning?topic=${encodeURIComponent(topic)}` : "/learning";

export default function FarmingLearningHub({ className, children }) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState(null);
  const dialog = useRef(null);
  const trigger = useRef(null);
  const searchInput = useRef(null);
  const title = useRef(null);
  const matches = farmingGuides.filter((guide) =>
    guide.name.toLowerCase().includes(search.trim().toLowerCase()),
  );

  useEffect(() => {
    if (!open) return;
    const modal = dialog.current;
    const previousOverflow = document.body.style.overflow;
    modal.showModal();
    document.body.style.overflow = "hidden";
    return () => {
      modal.close();
      document.body.style.overflow = previousOverflow;
      trigger.current?.focus();
    };
  }, [open]);

  useEffect(() => {
    if (open) (selected ? title.current : searchInput.current)?.focus();
  }, [open, selected]);

  function showHub() {
    setSearch("");
    setSelected(null);
    setOpen(true);
  }

  function closeOutside(event) {
    if (event.target !== dialog.current) return;
    const rect = dialog.current.getBoundingClientRect();
    if (
      event.clientX < rect.left ||
      event.clientX > rect.right ||
      event.clientY < rect.top ||
      event.clientY > rect.bottom
    )
      setOpen(false);
  }

  return (
    <>
      <button
        ref={trigger}
        type="button"
        className={className}
        id="openLearnHub"
        aria-haspopup="dialog"
        onClick={showHub}
      >
        {children}
      </button>
      <dialog
        ref={dialog}
        className="learn-modal"
        id="learnModal"
        aria-labelledby="learnModalTitle"
        onCancel={() => setOpen(false)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            setOpen(false);
          }
        }}
        onClick={closeOutside}
      >
        <div className="learn-modal-panel">
          <button
            type="button"
            className="learn-modal-close"
            id="learnModalClose"
            aria-label="Close learning hub"
            onClick={() => setOpen(false)}
          >
            ×
          </button>
          {selected && (
            <button
              type="button"
              className="learn-back"
              id="learnBackBtn"
              onClick={() => setSelected(null)}
            >
              ← All topics
            </button>
          )}
          <p className="eyebrow">
            {selected ? `${selected.icon} Farming guide` : "Learning hub"}
          </p>
          <h2 id="learnModalTitle" ref={title} tabIndex={-1}>
            {selected ? selected.name : "Learn farming"}
          </h2>
          {selected ? (
            <div id="learnLessonView">
              {selected.sections.map(([heading, content]) => (
                <section className="learn-lesson-section" key={heading}>
                  <h3>{heading}</h3>
                  <p>{content}</p>
                </section>
              ))}
              <p className="learn-hub-ai">
                <a href={learningURL(selected.name)}>
                  Ask the AI Learning Assistant about it →
                </a>
              </p>
            </div>
          ) : (
            <div id="learnHubView">
              <p className="learn-modal-subtitle">
                Pick a topic to start learning, or search for one below.
              </p>
              <p className="learn-hub-ai">
                <a href="/learning">Ask the AI Learning Assistant about it →</a>
              </p>
              <div className="learn-search">
                <input
                  ref={searchInput}
                  type="search"
                  id="learnSearchInput"
                  placeholder="Search e.g. goat farming, beekeeping..."
                  aria-label="Search farming topics"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                />
              </div>
              {matches.length ? (
                <div className="learn-grid" id="learnGrid">
                  {matches.map((guide) => (
                    <button
                      type="button"
                      className="learn-topic-card"
                      key={guide.id}
                      data-id={guide.id}
                      onClick={() => setSelected(guide)}
                    >
                      <span className="learn-topic-icon" aria-hidden="true">
                        {guide.icon}
                      </span>
                      <span>
                        <strong>{guide.name}</strong>
                        <small>{guide.sections.length} lessons</small>
                      </span>
                    </button>
                  ))}
                </div>
              ) : (
                <div className="learn-empty" id="learnEmptyState" role="status">
                  <p>We don’t have a guide for that yet.</p>
                  <a href={learningURL(search.trim())}>
                    Ask the AI Learning Assistant about it →
                  </a>
                </div>
              )}
            </div>
          )}
        </div>
      </dialog>
    </>
  );
}
