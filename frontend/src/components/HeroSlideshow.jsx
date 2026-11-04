import { farmingPhotos } from "../data/farmingPhotos.js";
import useRotatingImage from "../hooks/useRotatingImage.js";
import PhotoCredit from "./PhotoCredit.jsx";
import PhotoControls from "./PhotoControls.jsx";

export default function HeroSlideshow({
  imageId = "hero-slider",
  initialIndex = 0,
}) {
  const slideshow = useRotatingImage(farmingPhotos, initialIndex);
  const { photo, index } = slideshow;

  return (
    <div className="hero-image">
      <figure className="hero-photo" aria-label="Farming and produce photos">
        <div className="hero-photo-frame">
          <img
            key={index}
            id={imageId}
            src={photo.src}
            alt={photo.alt}
            style={{ objectPosition: photo.position }}
            loading="eager"
            width="600"
            height="500"
          />
          <PhotoControls slideshow={slideshow} imageId={imageId} />
        </div>
        <figcaption className="farming-photo-caption">
          <div className="farming-photo-heading">
            <span className="farming-photo-label">{photo.label}</span>
            <span className="farming-photo-position">
              {index + 1} / {farmingPhotos.length}
            </span>
          </div>
          <PhotoCredit photo={photo} />
        </figcaption>
      </figure>
    </div>
  );
}
