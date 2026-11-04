export default function PhotoControls({ slideshow, imageId, overlay = false }) {
  return (
    <div
      className={`photo-controls${overlay ? " photo-controls--overlay" : ""}`}
    >
      <button
        type="button"
        className="photo-control"
        aria-label="Previous photo"
        aria-controls={imageId}
        onClick={() => slideshow.move(-1)}
      >
        <span aria-hidden="true">←</span>
      </button>
      {!slideshow.reducedMotion && (
        <button
          type="button"
          className={`photo-control ${overlay ? "farming-auth-toggle" : "slideshow-toggle"}`}
          aria-controls={imageId}
          onClick={slideshow.togglePause}
        >
          {slideshow.isPaused ? "Play slideshow" : "Pause slideshow"}
        </button>
      )}
      <button
        type="button"
        className="photo-control"
        aria-label="Next photo"
        aria-controls={imageId}
        onClick={() => slideshow.move(1)}
      >
        <span aria-hidden="true">→</span>
      </button>
    </div>
  );
}
