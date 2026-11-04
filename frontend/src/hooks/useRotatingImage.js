import { useEffect, useState } from "react";

export default function useRotatingImage(images, paused = false) {
  const [index, setIndex] = useState(0);
  useEffect(() => {
    const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
    let timer;
    const update = () => {
      clearInterval(timer);
      if (!paused && !preference.matches)
        timer = setInterval(
          () => setIndex((value) => (value + 1) % images.length),
          4000,
        );
    };
    update();
    preference.addEventListener("change", update);
    return () => {
      clearInterval(timer);
      preference.removeEventListener("change", update);
    };
  }, [images, paused]);
  useEffect(() => {
    const next = images[(index + 1) % images.length];
    const preload = new Image();
    preload.src = next.src || next;
  }, [images, index]);
  return images[index];
}
