// AbortSignal.any is unavailable on older supported iPhones. Cancel a request
// when either its caller or its page is disposed, then release both listeners.
export async function fetchWithPageSignal(fetch, input, init, pageSignal) {
  if (!init.signal || init.signal === pageSignal)
    return fetch(input, { ...init, signal: pageSignal });

  const controller = new AbortController();
  const cleanups = [];
  try {
    for (const signal of [pageSignal, init.signal]) {
      if (signal.aborted) {
        controller.abort(signal.reason);
        break;
      }
      const abort = () => controller.abort(signal.reason);
      signal.addEventListener("abort", abort, { once: true });
      cleanups.push(() => signal.removeEventListener("abort", abort));
    }
    return await fetch(input, { ...init, signal: controller.signal });
  } finally {
    cleanups.forEach((cleanup) => cleanup());
  }
}
