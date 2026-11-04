import { useEffect, useState } from "react";

export default function useRotatingImage(images, initialIndex = 0) {
  const [index, setIndex] = useState(initialIndex);
  const [isPaused, setIsPaused] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(
    () =>
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches,
  );
  useEffect(() => {
    const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = (event) => setReducedMotion(event.matches);
    preference.addEventListener("change", update);
    return () => preference.removeEventListener("change", update);
  }, []);
  useEffect(() => {
    if (isPaused || reducedMotion) return;
    const timer = window.setInterval(
      () => setIndex((value) => (value + 1) % images.length),
      4000,
    );
    return () => window.clearInterval(timer);
  }, [images, isPaused, reducedMotion]);
  useEffect(() => {
    const next = images[(index + 1) % images.length];
    const preload = new Image();
    preload.src = next.src || next;
  }, [images, index]);
  return {
    photo: images[index],
    index,
    isPaused,
    reducedMotion,
    togglePause: () => setIsPaused((value) => !value),
    move: (step) => {
      setIndex((value) => (value + step + images.length) % images.length);
      setIsPaused(true);
    },
  };
}
