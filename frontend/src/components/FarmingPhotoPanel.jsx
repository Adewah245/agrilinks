import { farmingPhotos } from "../data/farmingPhotos.js";
import useRotatingImage from "../hooks/useRotatingImage.js";
import PhotoCredit from "./PhotoCredit.jsx";
import PhotoControls from "./PhotoControls.jsx";

export default function FarmingPhotoPanel({ children, imageId }) {
  const slideshow = useRotatingImage(farmingPhotos);
  const { photo } = slideshow;

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
        <PhotoControls slideshow={slideshow} imageId={imageId} overlay />
      </div>
    </section>
  );
}
