import { useState } from "react";
import { farmingPhotos } from "../data/farmingPhotos.js";
import useRotatingImage from "../hooks/useRotatingImage.js";
import PhotoCredit from "./PhotoCredit.jsx";

export default function FarmingPhotoPanel({ children, imageId }) {
  const [paused, setPaused] = useState(false);
  const photo = useRotatingImage(farmingPhotos, paused);

  return (
    <section className="auth-left farming-auth-panel">
      <img
        id={imageId}
        className="farming-auth-image"
        src={photo.src}
        alt={photo.alt}
        style={{ objectPosition: photo.position }}
      />
      <div className="farming-auth-overlay" />
      <div className="farming-auth-content">{children}</div>
      <div className="farming-auth-caption">
        <span className="farming-photo-label">{photo.label}</span>
        <PhotoCredit photo={photo} />
        <button
          type="button"
          className="farming-auth-toggle"
          aria-controls={imageId}
          onClick={() => setPaused((value) => !value)}
        >
          {paused ? "Play slideshow" : "Pause slideshow"}
        </button>
      </div>
    </section>
  );
}
