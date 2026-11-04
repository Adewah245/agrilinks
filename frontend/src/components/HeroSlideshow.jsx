import { useEffect, useState } from "react";
import { farmingPhotos as images } from "../data/farmingPhotos.js";
import PhotoCredit from "./PhotoCredit.jsx";

export default function HeroSlideshow() {
  const [currentImage, setCurrentImage] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(
    () =>
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches,
  );

  useEffect(() => {
    const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
    const updatePreference = (event) => setReducedMotion(event.matches);
    preference.addEventListener("change", updatePreference);
    return () => preference.removeEventListener("change", updatePreference);
  }, []);

  useEffect(() => {
    // Load the next photograph before it is displayed.
    const nextImage = new Image();
    nextImage.src = images[(currentImage + 1) % images.length].src;
  }, [currentImage]);

  useEffect(() => {
    if (isPaused || reducedMotion) return;

    const interval = window.setInterval(() => {
      setCurrentImage((current) => (current + 1) % images.length);
    }, 4000);

    return () => window.clearInterval(interval);
  }, [isPaused, reducedMotion]);

  const photo = images[currentImage];

  return (
    <div className="hero-image">
      <figure className="hero-photo">
        <div className="hero-photo-frame">
          <img
            key={currentImage}
            id="hero-slider"
            src={photo.src}
            alt={photo.alt}
            style={{ objectPosition: photo.position }}
            loading="eager"
            width="600"
            height="500"
          />
          {!reducedMotion && (
            <button
              type="button"
              className="slideshow-toggle"
              aria-controls="hero-slider"
              onClick={() => setIsPaused((paused) => !paused)}
            >
              {isPaused ? "Play slideshow" : "Pause slideshow"}
            </button>
          )}
        </div>
        <figcaption className="farming-photo-caption">
          <span className="farming-photo-label">{photo.label}</span>
          <PhotoCredit photo={photo} />
        </figcaption>
      </figure>
    </div>
  );
}
