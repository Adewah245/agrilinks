export default function PhotoCredit({ photo }) {
  return (
    <small className="farming-photo-credit">
      Photo:{" "}
      <a href={photo.source} target="_blank" rel="noreferrer">
        {photo.author}
      </a>
      {" · "}
      {photo.licenseUrl ? (
        <a href={photo.licenseUrl} target="_blank" rel="noreferrer">
          {photo.license}
        </a>
      ) : (
        <span>{photo.license}</span>
      )}
      {" · resized"}
    </small>
  );
}
