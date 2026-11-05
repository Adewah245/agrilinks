import { useEffect } from "react";

// Browser integrations are mounted after React commits a complete page. Each
// route uses document navigation, so forms retain Go's redirects and sessions.
// All timers and document/window listeners belong to this page's lifetime.
export function usePageBehaviors(mounts, data) {
  useEffect(() => {
    const controller = new AbortController();
    const cleanups = [];
    const timers = new Set();
    const frames = new Set();
    let active = true;
    const schedule =
      (repeat) =>
      (callback, delay, ...args) => {
        const id = window[repeat ? "setInterval" : "setTimeout"](() => {
          if (!repeat) timers.delete(id);
          if (active) callback(...args);
        }, delay);
        timers.add(id);
        return id;
      };
    const scope = {
      cleanup: (callback) => cleanups.push(callback),
      ready: (callback) => callback(),
      setTimeout: schedule(false),
      setInterval: schedule(true),
      requestAnimationFrame: (callback) => {
        const id = window.requestAnimationFrame((time) => {
          frames.delete(id);
          if (active) callback(time);
        });
        frames.add(id);
        return id;
      },
      fetch: (input, init = {}) =>
        window.fetch(input, {
          ...init,
          signal: init.signal
            ? AbortSignal.any([init.signal, controller.signal])
            : controller.signal,
        }),
    };
    const observe = (Constructor) =>
      Constructor &&
      class extends Constructor {
        constructor(...args) {
          super(...args);
          cleanups.push(() => this.disconnect());
        }
      };
    scope.ResizeObserver = observe(window.ResizeObserver);
    scope.IntersectionObserver = observe(window.IntersectionObserver);
    const scopedTarget = (target) =>
      new Proxy(target, {
        get(object, property) {
          if (property === "addEventListener")
            return (type, listener, options) => {
              object.addEventListener(type, listener, {
                ...(typeof options === "boolean"
                  ? { capture: options }
                  : options),
                signal: controller.signal,
              });
            };
          if (property in scope) return scope[property];
          const value = Reflect.get(object, property, object);
          // Preserve Web constructors and their static helpers. Binding
          // MediaRecorder or URL drops isTypeSupported/createObjectURL.
          if (typeof value === "function" && /^[A-Z]/.test(String(property)))
            return value;
          return typeof value === "function" ? value.bind(object) : value;
        },
        set(object, property, value) {
          return Reflect.set(object, property, value, object);
        },
      });
    scope.document = scopedTarget(document);
    scope.window = scopedTarget(window);
    for (const mount of mounts) {
      const cleanup = mount(scope, data);
      if (typeof cleanup === "function") cleanups.push(cleanup);
    }
    return () => {
      active = false;
      controller.abort();
      timers.forEach((id) => {
        window.clearTimeout(id);
        window.clearInterval(id);
      });
      frames.forEach((id) => window.cancelAnimationFrame(id));
      cleanups.reverse().forEach((cleanup) => cleanup());
    };
    // Pages receive one immutable server snapshot per document navigation.
  }, []);
}
