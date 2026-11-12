// The native entry point installs this adapter. Ordinary web pages continue
// using browser navigation and device APIs.
export const mobileApp = () => globalThis.window?.agrilinksApp;
export const hasNativeCamera = () => Boolean(mobileApp()?.captureSelfie);
export const hasDeviceLocation = () =>
  Boolean(mobileApp()?.locate || globalThis.navigator?.geolocation);

export function navigateTo(url) {
  if (mobileApp()) return mobileApp().navigate(url);
  window.location.assign(url);
}

export function reloadPage() {
  if (mobileApp()) return mobileApp().refresh();
  window.location.reload();
}

export function getDeviceLocation(options = {}) {
  if (mobileApp()?.locate) return mobileApp().locate(options);
  return new Promise((resolve, reject) => {
    if (!globalThis.navigator?.geolocation) {
      reject(
        Object.assign(
          new Error("This device does not support location access."),
          { code: 2 },
        ),
      );
      return;
    }
    navigator.geolocation.getCurrentPosition(resolve, reject, options);
  });
}

export function captureNativeSelfie() {
  return mobileApp().captureSelfie();
}
