export default function VoiceControls({ prefix, showInput = true }) {
  return (
    <div className="voice-controls" id={`${prefix}-voice`}>
      <div className="voice-buttons">
        {showInput && (
          <button
            type="button"
            id={`${prefix}-voice-start`}
            className="voice-question-button"
            aria-pressed="false"
          >
            🎙️ Ask by voice
          </button>
        )}
        <button type="button" id={`${prefix}-voice-stop`} hidden>
          Finish recording
        </button>
        <button type="button" id={`${prefix}-speech-stop`} hidden>
          Stop reading
        </button>
      </div>
      <label className="voice-read-option">
        <input type="checkbox" id={`${prefix}-read-aloud`} /> Read answers aloud
      </label>
      <p
        id={`${prefix}-voice-status`}
        className="voice-status"
        role="status"
      ></p>
    </div>
  );
}
