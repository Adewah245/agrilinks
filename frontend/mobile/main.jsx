import { createRoot } from "react-dom/client";
import { Capacitor, CapacitorHttp } from "@capacitor/core";
import { App } from "@capacitor/app";
import { AppLauncher } from "@capacitor/app-launcher";
import { Browser } from "@capacitor/browser";
import {
  Camera,
  CameraDirection,
  CameraResultType,
  CameraSource,
} from "@capacitor/camera";
import { Geolocation } from "@capacitor/geolocation";
import { Keyboard } from "@capacitor/keyboard";
import { Network } from "@capacitor/network";
import { SplashScreen } from "@capacitor/splash-screen";
import { StatusBar, Style } from "@capacitor/status-bar";
import { pages } from "../src/pages/registry.js";
import MobileApp from "./MobileApp.jsx";
import { backendOrigin, createURLPolicy } from "./lib/url-policy.js";
import { createNativeFetch, createPreviewFetch } from "./lib/http.js";
import { createMobileRuntime } from "./lib/runtime.js";
import { createStyleLoader } from "./lib/assets.js";
import "../src/styles/index.css";
import "./mobile.css";

const native = Capacitor.isNativePlatform();
const platform = Capacitor.getPlatform();
const origin = backendOrigin(import.meta.env.VITE_MOBILE_BACKEND_URL, {
  production: import.meta.env.MODE === "production",
  platform,
});
const localURL = window.location.href;
const webFetch = window.fetch.bind(window);
const policy = createURLPolicy(origin, localURL);
const fetch = native
  ? createNativeFetch({
      origin,
      localURL,
      platform,
      request: (options) => CapacitorHttp.request(options),
      webFetch,
    })
  : createPreviewFetch({ origin, localURL, webFetch });
let deviceOperations = 0;

async function openURL(value) {
  const url = new URL(value);
  if (["http:", "https:"].includes(url.protocol)) {
    if (native) await Browser.open({ url: url.href });
    else window.open(url.href, "_blank", "noopener,noreferrer");
  } else if (["tel:", "mailto:", "sms:"].includes(url.protocol)) {
    if (native) await AppLauncher.openUrl({ url: url.href });
    else window.location.assign(url.href);
  } else throw new Error("This link cannot be opened.");
}

async function captureSelfie() {
  deviceOperations++;
  try {
    const photo = await Camera.getPhoto({
      source: CameraSource.Camera,
      direction: CameraDirection.Front,
      resultType: CameraResultType.Uri,
      quality: 85,
      width: 1024,
      height: 1024,
      allowEditing: false,
      saveToGallery: false,
    });
    const response = await webFetch(
      photo.webPath || Capacitor.convertFileSrc(photo.path),
    );
    const blob = await response.blob();
    if (
      !blob.size ||
      blob.size > 2 * 1024 * 1024 ||
      !blob.type.startsWith("image/")
    )
      throw new Error("Could not save your selfie. Please take it again.");
    return blob;
  } finally {
    deviceOperations--;
  }
}

async function locate(options) {
  deviceOperations++;
  try {
    let permission = await Geolocation.checkPermissions();
    if (["prompt", "prompt-with-rationale"].includes(permission.location))
      permission = await Geolocation.requestPermissions({
        permissions: ["location"],
      });
    if (
      permission.location === "denied" &&
      permission.coarseLocation !== "granted"
    )
      throw Object.assign(
        new Error(
          "Allow location access for Agrilinks in your device settings.",
        ),
        { code: 1 },
      );
    return await Geolocation.getCurrentPosition(options);
  } catch (error) {
    if (typeof error.code === "number") throw error;
    const denied = /denied|permission|authoriz/i.test(error.message || "");
    const timedOut = /timeout|timed out/i.test(error.message || "");
    throw Object.assign(
      new Error(
        denied
          ? "Allow location access for Agrilinks in your device settings."
          : timedOut
            ? "Could not find your location in time. Try again outdoors."
            : "Your location is unavailable. Check that location services are enabled.",
      ),
      { code: denied ? 1 : timedOut ? 3 : 2 },
    );
  } finally {
    deviceOperations--;
  }
}

const runtime = createMobileRuntime({
  fetch,
  policy,
  openURL,
  hasPage: (name) => Object.hasOwn(pages, name),
  loadStyles: createStyleLoader(),
  captureSelfie: native ? captureSelfie : undefined,
  locate: native ? locate : undefined,
});
const cleanup = runtime.install();
createRoot(document.getElementById("root")).render(
  <MobileApp runtime={runtime} />,
);

async function start() {
  let path =
    native || window.location.pathname === "/"
      ? "/dashboard"
      : `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (native) {
    await Promise.allSettled([
      StatusBar.setStyle({ style: Style.Light }),
      platform === "android"
        ? StatusBar.setBackgroundColor({ color: "#f7f9f4" })
        : Promise.resolve(),
      SplashScreen.hide(),
      App.addListener("backButton", ({ canGoBack }) => {
        if (runtime.getState().busy) return;
        if (runtime.getState().loading && runtime.getState().page) {
          runtime.cancelNavigation();
          return;
        }
        if (canGoBack || Number(window.history.state?.agrilinksIndex || 0) > 0)
          window.history.back();
        else App.minimizeApp();
      }),
      App.addListener("appUrlOpen", async ({ url }) => {
        try {
          const destination = policy.deepLink(url).path;
          await Browser.close().catch(() => {});
          await runtime.navigate(destination);
        } catch {
          /* Ignore links belonging to other apps. */
        }
      }),
      App.addListener("appRestoredResult", ({ pluginId }) => {
        if (pluginId === "Camera")
          runtime.showNotice(
            "Please take your selfie again to finish verification.",
          );
      }),
      App.addListener("appStateChange", ({ isActive }) => {
        if (
          isActive &&
          !deviceOperations &&
          runtime.getState().path.startsWith("/wallet")
        )
          runtime.refreshWhenSafe();
      }),
      Browser.addListener("browserFinished", () => runtime.refreshWhenSafe()),
      Network.addListener("networkStatusChange", ({ connected }) =>
        runtime.showConnection(connected),
      ),
      Keyboard.addListener("keyboardDidShow", () =>
        document.documentElement.classList.add("mobile-keyboard-open"),
      ),
      Keyboard.addListener("keyboardDidHide", () =>
        document.documentElement.classList.remove("mobile-keyboard-open"),
      ),
    ]);
    const launch = await App.getLaunchUrl().catch(() => null);
    if (launch?.url) {
      try {
        path = policy.deepLink(launch.url).path;
      } catch {
        /* Use the normal signed-in landing page. */
      }
    }
  }
  runtime.navigate(path, { replace: true });
}
start();
if (import.meta.hot) import.meta.hot.dispose(cleanup);
