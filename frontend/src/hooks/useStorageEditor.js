import { useEffect, useRef, useState } from "react";

export default function useStorageEditor(initialProduct) {
  const [product, setProduct] = useState(initialProduct || null);
  const [open, setOpen] = useState(Boolean(initialProduct));
  const [status, setStatus] = useState("");
  const form = useRef(null);
  const request = useRef(0);

  function close() {
    request.current++;
    setOpen(false);
  }
  function add() {
    request.current++;
    setProduct(null);
    setStatus("");
    setOpen(true);
  }

  useEffect(() => {
    if (!open) return;
    const escape = (event) => {
      if (event.key === "Escape") close();
    };
    document.addEventListener("keydown", escape);
    form.current?.querySelector('input:not([type="hidden"])')?.focus();
    return () => {
      request.current++;
      document.removeEventListener("keydown", escape);
    };
  }, [open, product]);

  function capture() {
    if (!navigator.geolocation) {
      setStatus("This device does not support GPS.");
      return;
    }
    const id = ++request.current;
    setStatus("Getting your farm location…");
    navigator.geolocation.getCurrentPosition(
      (position) => {
        if (id !== request.current || !form.current) return;
        const fields = form.current.elements;
        fields.latitude.value = position.coords.latitude;
        fields.longitude.value = position.coords.longitude;
        fields.location_accuracy.value = position.coords.accuracy || "";
        setStatus(
          `Location captured (accuracy ${Math.round(position.coords.accuracy)} m)`,
        );
      },
      () => {
        if (id === request.current)
          setStatus("Allow location access to register this product.");
      },
      { enableHighAccuracy: true, timeout: 15000, maximumAge: 30000 },
    );
  }

  function submit(event) {
    if (
      !form.current.elements.latitude.value ||
      !form.current.elements.longitude.value
    ) {
      event.preventDefault();
      capture();
    }
  }
  return { product, open, status, form, close, add, capture, submit };
}
