import { useEffect, useRef, useState } from "react";

export default function WithdrawalEvidenceCapture({ quote }) {
  const root = useRef(null);
  const video = useRef(null);
  const stream = useRef(null);
  const challenge = useRef("");
  const mounted = useRef(true);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [token, setToken] = useState("");
  const [message, setMessage] = useState("");
  const [consent, setConsent] = useState(false);
  const stop = () => {
    stream.current?.getTracks().forEach((track) => track.stop());
    stream.current = null;
  };
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      stop();
    };
  }, []);
  useEffect(() => {
    const form = root.current.closest("form");
    const check = (event) => {
      if (!token) {
        event.preventDefault();
        setMessage(
          "Capture and save your private selfie and location before confirming.",
        );
        root.current.querySelector("button").focus();
      }
    };
    // Run before the wallet's duplicate-submit guard, including after a saved
    // token expires or consent is withdrawn and this listener is reattached.
    form.addEventListener("submit", check, true);
    return () => form.removeEventListener("submit", check, true);
  }, [token]);
  useEffect(() => {
    if (!token) return;
    const timer = setTimeout(
      () => {
        setToken("");
        setMessage(
          "Your evidence capture expired. Capture it again before submitting.",
        );
      },
      9 * 60 * 1000,
    );
    return () => clearTimeout(timer);
  }, [token]);
  async function json(response) {
    const data = await response.json();
    if (!response.ok)
      throw new Error(
        data.error || "Could not save private evidence. Try again.",
      );
    return data;
  }
  async function start() {
    setBusy(true);
    setToken("");
    setReady(false);
    stop();
    try {
      if (!consent)
        throw new Error(
          "Confirm that you agree to this private verification first.",
        );
      if (!navigator.mediaDevices?.getUserMedia || !navigator.geolocation)
        throw new Error(
          "Use a browser with camera and location access to request a withdrawal.",
        );
      const result = await json(
        await fetch("/wallet/evidence/challenge", {
          method: "POST",
          body: new URLSearchParams({ quote }),
        }),
      );
      challenge.current = result.token;
      if (!mounted.current) return;
      const media = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "user", width: { ideal: 960 } },
        audio: false,
      });
      if (!mounted.current) {
        media.getTracks().forEach((track) => track.stop());
        return;
      }
      stream.current = media;
      video.current.srcObject = media;
      await video.current.play();
      setMessage(
        "Keep your face clearly visible, then capture your selfie and location.",
      );
    } catch (error) {
      stop();
      setMessage(
        error.name === "NotAllowedError"
          ? "Allow camera access in your browser to continue."
          : error.message,
      );
    } finally {
      setBusy(false);
    }
  }
  async function capture() {
    setBusy(true);
    setMessage("Capturing your current location and saving private evidence…");
    try {
      const position = await new Promise((resolve, reject) =>
        navigator.geolocation.getCurrentPosition(resolve, reject, {
          enableHighAccuracy: true,
          maximumAge: 0,
          timeout: 20000,
        }),
      );
      if (!mounted.current) return;
      const canvas = document.createElement("canvas");
      canvas.width = video.current.videoWidth;
      canvas.height = video.current.videoHeight;
      canvas.getContext("2d").drawImage(video.current, 0, 0);
      const blob = await new Promise((resolve) =>
        canvas.toBlob(resolve, "image/jpeg", 0.85),
      );
      if (!blob)
        throw new Error(
          "Could not capture the selfie. Open your camera again.",
        );
      const body = new FormData();
      body.set("selfie", blob, "selfie.jpg");
      body.set("token", challenge.current);
      body.set("latitude", String(position.coords.latitude));
      body.set("longitude", String(position.coords.longitude));
      body.set("accuracy", String(position.coords.accuracy));
      body.set("captured_at", new Date(position.timestamp).toISOString());
      await json(await fetch("/wallet/evidence", { method: "POST", body }));
      setToken(challenge.current);
      setMessage(
        "Private evidence saved. Your withdrawal will wait for an administrator's review.",
      );
      stop();
      setReady(false);
    } catch (error) {
      setMessage(
        error.code === 1
          ? "Allow location access to continue. Your location is shared only for private verification."
          : error.message ||
              "Could not capture your location. Try again outdoors.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <fieldset ref={root} className="withdrawal-evidence">
      <legend>Private withdrawal verification</legend>
      <p>
        A finance administrator will review your selfie and current location
        before any payout is submitted. These are private verification records,
        separate from your public profile.
      </p>
      <label>
        <input
          type="checkbox"
          checked={consent}
          disabled={busy}
          onChange={(e) => {
            setConsent(e.target.checked);
            if (!e.target.checked) {
              setToken("");
              stop();
              setReady(false);
            }
          }}
          required
        />{" "}
        I agree to capture my selfie and location for this withdrawal review.
      </label>
      <input type="hidden" name="evidence_token" value={token} />
      <video
        ref={video}
        className="evidence-camera"
        muted
        playsInline
        onLoadedMetadata={() => setReady(true)}
        hidden={!stream.current}
      />
      <div className="evidence-actions">
        <button type="button" onClick={start} disabled={busy || !consent}>
          {token ? "Capture again" : "Open camera"}
        </button>
        <button
          type="button"
          onClick={capture}
          disabled={busy || !ready || !consent}
        >
          Capture selfie and location
        </button>
      </div>
      <p role="status" aria-live="polite">
        {message}
      </p>
    </fieldset>
  );
}
