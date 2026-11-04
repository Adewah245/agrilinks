// Presentation helpers shared by the page components. Wallet amounts remain
// decimal strings; financial calculations and authorization stay on the server.
export const truth = (value) =>
  Array.isArray(value) ? value.length > 0 : Boolean(value);
export const items = (value) =>
  Array.isArray(value) ? value : Object.values(value || {});
export const len = (value) => value?.length ?? Object.keys(value || {}).length;
export const withValue = (value, render, fallback = () => null) =>
  truth(value) ? render(value) : fallback();
export const text = (value) => (value == null ? "" : String(value));
export const urlquery = (value) => encodeURIComponent(text(value));
export const requestKey = () =>
  Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("");
export const hasCoordinates = (latitude, longitude) =>
  Boolean(latitude && longitude);

const number = (value, decimals) =>
  Number.isFinite(Number(value))
    ? Number(value).toLocaleString("en-US", {
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      })
    : "0";
export const money = (value) => number(value, 2);
export const money0 = (value) => number(value, 0);
export const comma = money0;

export function style(value) {
  return Object.fromEntries(
    value.split(";").flatMap((declaration) => {
      const colon = declaration.indexOf(":");
      if (colon < 0) return [];
      const key = declaration.slice(0, colon).trim();
      return [
        [
          key.startsWith("--")
            ? key
            : key.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase()),
          declaration.slice(colon + 1).trim(),
        ],
      ];
    }),
  );
}

function readable(value) {
  const label = text(value).trim().replaceAll("_", " ").toLowerCase();
  return label.charAt(0).toUpperCase() + label.slice(1);
}
const labels = (values) => (value) =>
  values[text(value).trim().toLowerCase()] || readable(value);
export const humanOrderStatus = labels({
  pending: "Waiting for payment",
  confirmed: "Order confirmed",
  processing: "Being prepared",
  ready_for_delivery: "Ready for delivery",
  shipped: "On the way",
  delivered: "Delivered",
  cancelled: "Cancelled",
  canceled: "Cancelled",
});
export const humanPaymentStatus = labels({
  pending: "Waiting",
  successful: "Paid",
  failed: "Payment failed",
  cancelled: "Cancelled",
  canceled: "Cancelled",
  refunded: "Refunded",
});
export const humanNegotiationStatus = labels({ rejected: "Declined" });
export const humanOfferStatus = labels({
  pending: "Waiting for reply",
  rejected: "Declined",
});
export const humanStatus = labels({
  pending: "Waiting",
  assigned: "Driver assigned",
  picked_up: "Picked up",
  in_transit: "On the way",
  failed: "Not successful",
});
export const humanBadge = readable;

export function printf(format, value) {
  return format
    .replace(
      /%(\+)?\.0f/g,
      (_, sign) =>
        (sign && Number(value) >= 0 ? "+" : "") + Number(value || 0).toFixed(0),
    )
    .replaceAll("%%", "%");
}

export function formatDate(value, format) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  // Match Go's UTC page timestamps consistently across browser time zones.
  const hour = date.getUTCHours();
  const pad = (value) => String(value).padStart(2, "0");
  const month = new Intl.DateTimeFormat("en-GB", {
    month: "short",
    timeZone: "UTC",
  }).format(date);
  const tokens = {
    2006: date.getUTCFullYear(),
    January: new Intl.DateTimeFormat("en-GB", {
      month: "long",
      timeZone: "UTC",
    }).format(date),
    Jan: month,
    "01": pad(date.getUTCMonth() + 1),
    "02": pad(date.getUTCDate()),
    2: date.getUTCDate(),
    15: pad(hour),
    "03": pad(hour % 12 || 12),
    3: hour % 12 || 12,
    "04": pad(date.getUTCMinutes()),
    "05": pad(date.getUTCSeconds()),
    PM: hour < 12 ? "AM" : "PM",
    MST: "UTC",
    "Z07:00": "Z",
  };
  return format.replace(
    /Z07:00|January|2006|Jan|MST|PM|01|02|15|03|04|05|2|3/g,
    (token) => tokens[token],
  );
}
