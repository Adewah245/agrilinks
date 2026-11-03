import { useEffect, useState } from 'react';

const images = Array.from(
  { length: 10 },
  (_, index) => `/static/assets/images/Farmer${index + 1}.jpeg`,
);

export default function HeroSlideshow() {
  const [currentImage, setCurrentImage] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(
    () => window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  );

  useEffect(() => {
    const preference = window.matchMedia('(prefers-reduced-motion: reduce)');
    const updatePreference = (event) => setReducedMotion(event.matches);
    preference.addEventListener('change', updatePreference);
    return () => preference.removeEventListener('change', updatePreference);
  }, []);

  useEffect(() => {
    // Load the next photograph before it is displayed.
    const nextImage = new Image();
    nextImage.src = images[(currentImage + 1) % images.length];
  }, [currentImage]);

  useEffect(() => {
    if (isPaused || reducedMotion) return;

    const interval = window.setInterval(() => {
      setCurrentImage((current) => (current + 1) % images.length);
    }, 4000);

    return () => window.clearInterval(interval);
  }, [isPaused, reducedMotion]);

  return (
    <div className="hero-image">
      <img
        key={currentImage}
        id="hero-slider"
        src={images[currentImage]}
        alt="Farmer working in agriculture"
        loading="eager"
        width="600"
        height="400"
      />
      {!reducedMotion && (
        <button
          type="button"
          className="slideshow-toggle"
          aria-controls="hero-slider"
          onClick={() => setIsPaused((paused) => !paused)}
        >
          {isPaused ? 'Play slideshow' : 'Pause slideshow'}
        </button>
      )}
    </div>
  );
}
