document.addEventListener("DOMContentLoaded", () => {
  const images = [
    "/static/assets/images/Farmer1.jpeg",
    "/static/assets/images/Farmer2.jpeg",
    "/static/assets/images/Farmer3.jpeg",
    "/static/assets/images/Farmer4.jpeg",
    "/static/assets/images/Farmer5.jpeg", // Duplication removed in loop logic
    "/static/assets/images/Farmer6.jpeg",
    "/static/assets/images/Farmer7.jpeg",
    "/static/assets/images/Farmer8.jpeg",
    "/static/assets/images/Farmer9.jpeg",
    "/static/assets/images/Farmer10.jpeg"
  ];

  // Remove potential duplicate entries from array dynamically
  const uniqueImages = [...new Set(images)];

  const slider = document.getElementById("hero-slider");
  if (!slider || uniqueImages.length <= 1) return;

  // Preload all slider images into memory to prevent network lag on swap
  uniqueImages.forEach((src) => {
    const img = new Image();
    img.src = src;
  });

  let currentImage = 0;

  function changeImage() {
    slider.style.opacity = "0";

    setTimeout(() => {
      currentImage = (currentImage + 1) % uniqueImages.length;
      slider.src = uniqueImages[currentImage];
      slider.style.opacity = "1";
    }, 400);
  }

  setInterval(changeImage, 4000);
});