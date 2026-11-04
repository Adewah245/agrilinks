export default function mount(scope, data) {
  const {
    document,
    window,
    setTimeout,
    setInterval,
    requestAnimationFrame,
    fetch,
    ResizeObserver,
    IntersectionObserver,
  } = scope;
  (() => {
    function initDeviceWeather() {
      const card = document.querySelector("[data-device-weather]");
      if (!card) return;
      const field = (name) => card.querySelector(`[data-weather-${name}]`);
      const status = field("status");
      const button = field("refresh");
      const values = field("values");
      const updated = field("updated");
      const refreshInterval = 5 * 60 * 1000;
      const maxAccuracy = 1000;
      let busy = false;
      let mayRefresh = false;
      let lastAttempt = 0;
      let generation = 0;
      let activeRequest;
      let activePlaceRequest;
      let detectedPlace = "";
      let weatherSource = "";
      const placeStatus = field("place-status");
      const placeCredit = field("location-credit");
      const placeCache = new Map();

      function showLocation() {
        const name = detectedPlace || "Your current location";
        field("location").textContent = weatherSource
          ? `${name} · ${weatherSource}`
          : name;
      }

      // This provider's free endpoint is browser-only and must receive the
      // current device coordinates. Never omit them or use its IP fallback.
      async function resolvePlace(latitude, longitude, currentGeneration) {
        const key = `${latitude.toFixed(4)},${longitude.toFixed(4)}`;
        const cached = placeCache.get(key);
        if (cached) {
          detectedPlace = cached;
          placeStatus.hidden = true;
          placeCredit.hidden = false;
          showLocation();
          return;
        }
        const controller = new AbortController();
        activePlaceRequest = controller;
        const timeout = setTimeout(() => controller.abort(), 6000);
        placeStatus.textContent = "Finding your place name…";
        placeStatus.hidden = false;
        try {
          const query = new URLSearchParams({
            latitude: String(latitude),
            longitude: String(longitude),
            localityLanguage: "en",
          });
          const response = await fetch(
            `https://api.bigdatacloud.net/data/reverse-geocode-client?${query}`,
            {
              credentials: "omit",
              referrerPolicy: "no-referrer",
              cache: "no-store",
              signal: controller.signal,
            },
          );
          if (!response.ok) throw new Error("Place lookup unavailable");
          const place = await response.json();
          if (currentGeneration !== generation) return;
          // A provider response for a different point (including an IP
          // estimate) must never label the device's weather location.
          if (
            (typeof place.lookupSource === "string" &&
              /ip/i.test(place.lookupSource)) ||
            !Number.isFinite(place.latitude) ||
            !Number.isFinite(place.longitude) ||
            Math.abs(place.latitude - latitude) > 0.0001 ||
            Math.abs(place.longitude - longitude) > 0.0001
          ) {
            throw new Error(
              "Place lookup did not match the device coordinates",
            );
          }
          const clean = (value) =>
            typeof value === "string" ? value.trim() : "";
          const locality = clean(place.city) || clean(place.locality);
          if (!locality) throw new Error("No locality name available");
          const names = [
            locality,
            clean(place.principalSubdivision),
            clean(place.countryName),
          ].filter(Boolean);
          const seen = new Set();
          detectedPlace = names
            .filter((name) => {
              const key = name.toLocaleLowerCase();
              if (seen.has(key)) return false;
              seen.add(key);
              return true;
            })
            .join(", ");
          if (placeCache.size >= 64)
            placeCache.delete(placeCache.keys().next().value);
          placeCache.set(key, detectedPlace);
          placeStatus.hidden = true;
          placeCredit.hidden = false;
          showLocation();
        } catch {
          if (currentGeneration !== generation) return;
          placeStatus.textContent =
            "Place name unavailable. Your weather still uses the detected coordinates.";
          placeStatus.hidden = false;
        } finally {
          clearTimeout(timeout);
          if (activePlaceRequest === controller) activePlaceRequest = undefined;
        }
      }

      function fail(message, state = "error") {
        values.hidden = true;
        updated.hidden = true;
        status.textContent = message;
        card.dataset.weatherState = state;
        card.setAttribute("aria-busy", "false");
        busy = false;
        button.disabled = false;
        button.textContent = "Try location again";
      }

      function locationError(error) {
        mayRefresh = false;
        switch (error.code) {
          case 1:
            fail(
              "Location access is blocked. Allow location in your browser’s site settings, then try again.",
              "denied",
            );
            break;
          case 3:
            fail(
              "Your device took too long to find your location. Turn on location services and try again.",
              "timeout",
            );
            break;
          default:
            fail(
              "Your device could not determine your location. Turn on location services and try again.",
              "unavailable",
            );
        }
      }

      async function loadForecast(position, currentGeneration) {
        if (currentGeneration !== generation) return;
        const { latitude, longitude, accuracy } = position.coords;
        if (
          !Number.isFinite(latitude) ||
          Math.abs(latitude) > 90 ||
          !Number.isFinite(longitude) ||
          Math.abs(longitude) > 180 ||
          !Number.isFinite(accuracy) ||
          accuracy < 0
        ) {
          mayRefresh = false;
          fail(
            "Your device did not provide a valid location. Please try again.",
            "unavailable",
          );
          return;
        }
        if (accuracy > maxAccuracy) {
          mayRefresh = false;
          const distance = `${(accuracy / 1000).toFixed(1)} km`;
          fail(
            `Your device’s location is only accurate to about ${distance}. Enable precise location and try again.`,
            "imprecise",
          );
          return;
        }
        if (
          !Number.isFinite(position.timestamp) ||
          Date.now() - position.timestamp > 60000
        ) {
          mayRefresh = false;
          fail(
            "Your device returned an old location. Please try again for your current position.",
            "unavailable",
          );
          return;
        }
        mayRefresh = true;
        resolvePlace(latitude, longitude, currentGeneration);
        card.dataset.weatherState = "loading";
        status.textContent = "Getting weather for your current location…";
        button.textContent = "Loading weather…";
        const controller = new AbortController();
        activeRequest = controller;
        const timeout = setTimeout(() => controller.abort(), 15000);
        try {
          const response = await fetch("/api/weather/current", {
            method: "POST",
            credentials: "same-origin",
            cache: "no-store",
            headers: {
              "Content-Type": "application/json",
              Accept: "application/json",
            },
            body: JSON.stringify({ latitude, longitude, accuracy }),
            signal: controller.signal,
          });
          if (currentGeneration !== generation) return;
          if (response.redirected) {
            mayRefresh = false;
            fail("Please sign in again to view weather for your location.");
            return;
          }
          const weather = await response.json();
          if (currentGeneration !== generation) return;
          if (!response.ok || !weather.available || weather.approximate) {
            fail(
              weather.error ||
                "Weather is temporarily unavailable for your current location. Please try again.",
            );
            return;
          }
          const observed = new Date(weather.last_updated);
          if (Number.isNaN(observed.getTime()))
            throw new Error("Missing observation time");
          weatherSource = weather.source;
          showLocation();
          field("temperature").textContent = weather.temperature;
          field("summary").textContent = weather.summary;
          field("icon").textContent = weather.icon;
          field("humidity").textContent = weather.humidity;
          field("rain").textContent = weather.rain_chance;
          field("conditions").textContent = weather.conditions;
          field("recommendation").textContent = weather.recommendation;
          field("accuracy").textContent =
            `Device location accuracy: about ${Math.max(1, Math.round(accuracy))} m`;
          const condition = weather.stale
            ? "Older conditions"
            : weather.cached
              ? "Cached conditions"
              : "Current conditions";
          const time = new Intl.DateTimeFormat(undefined, {
            day: "numeric",
            month: "short",
            hour: "numeric",
            minute: "2-digit",
            timeZoneName: "short",
          }).format(observed);
          updated.textContent = `${condition} · Updated ${time}`;
          updated.hidden = false;
          values.hidden = false;
          status.textContent =
            weather.error || "Using your device’s current location.";
          card.dataset.weatherState = weather.stale ? "stale" : "ready";
          card.setAttribute("aria-busy", "false");
          busy = false;
          button.disabled = false;
          button.textContent = "Refresh location & weather";
        } catch (error) {
          if (currentGeneration !== generation) return;
          fail(
            error.name === "AbortError"
              ? "The weather request took too long. Please try again."
              : "Could not load weather for your current location. Check your connection and try again.",
          );
        } finally {
          clearTimeout(timeout);
          if (activeRequest === controller) activeRequest = undefined;
        }
      }

      function requestLocation() {
        if (busy) return;
        const currentGeneration = ++generation;
        activePlaceRequest?.abort();
        detectedPlace = "";
        weatherSource = "";
        placeStatus.hidden = true;
        placeCredit.hidden = true;
        field("location").textContent = "Your device location";
        lastAttempt = Date.now();
        if (!window.isSecureContext) {
          fail(
            "Device location needs a secure connection. Open the HTTPS version of this site and try again.",
            "insecure",
          );
          return;
        }
        if (!navigator.geolocation) {
          fail(
            "This browser cannot share your device location. Use a browser with location support.",
            "unsupported",
          );
          return;
        }
        busy = true;
        values.hidden = true;
        updated.hidden = true;
        button.disabled = true;
        button.textContent = "Finding your location…";
        card.dataset.weatherState = "locating";
        card.setAttribute("aria-busy", "true");
        status.textContent =
          "Finding your current location. Allow location access when your browser asks.";
        try {
          navigator.geolocation.getCurrentPosition(
            (position) => loadForecast(position, currentGeneration),
            (error) => {
              if (currentGeneration === generation) locationError(error);
            },
            { enableHighAccuracy: true, timeout: 15000, maximumAge: 0 },
          );
        } catch {
          locationError({ code: 2 });
        }
      }

      button.addEventListener("click", requestLocation);
      const refresh = () => {
        if (
          !document.hidden &&
          mayRefresh &&
          Date.now() - lastAttempt >= refreshInterval
        )
          requestLocation();
      };
      let timer = setInterval(refresh, refreshInterval);
      document.addEventListener("visibilitychange", refresh);
      window.addEventListener("pagehide", () => {
        ++generation;
        activeRequest?.abort();
        activePlaceRequest?.abort();
        clearInterval(timer);
        busy = false;
      });
      window.addEventListener("pageshow", (event) => {
        if (event.persisted) {
          mayRefresh = false;
          requestLocation();
          timer = setInterval(refresh, refreshInterval);
        }
      });
      requestLocation();
    }
    initDeviceWeather();
  })();
}
