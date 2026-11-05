export function addExampleButton(
  scope,
  container,
  payload,
  endpoint = "/learning/example",
) {
  const { document, fetch } = scope;
  const button = document.createElement("button");
  button.type = "button";
  button.className = "ai-example-button";
  button.textContent = "Show an example picture";
  const status = document.createElement("p");
  status.className = "example-image-status";
  status.setAttribute("role", "status");
  container.append(button, status);
  const controller = new AbortController();
  let shown = false;
  button.addEventListener("click", async () => {
    if (shown || button.disabled) return;
    button.disabled = true;
    status.textContent =
      "Making an example picture… You can still read or listen to the answer.";
    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(payload),
        signal: controller.signal,
      });
      const result = await response.json();
      if (controller.signal.aborted) return;
      if (!response.ok)
        throw new Error(
          result.error ||
            "The example picture is unavailable. Please try again.",
        );
      if (
        typeof result.data_url !== "string" ||
        result.data_url.length > 16 * 1024 * 1024 ||
        !/^data:image\/(png|jpeg|webp);base64,[A-Za-z0-9+/]+=*$/.test(
          result.data_url,
        )
      )
        throw new Error(
          "The example picture could not be shown. Please try again.",
        );
      const figure = document.createElement("figure");
      figure.className = "ai-example-picture";
      const image = document.createElement("img");
      image.alt = result.alt || "Example farming picture";
      image.src = result.data_url;
      const caption = document.createElement("figcaption");
      caption.textContent =
        result.caption || "Example picture to help explain the answer.";
      figure.append(image, caption);
      container.append(figure);
      shown = true;
      button.textContent = "Example picture shown";
      status.textContent = "";
    } catch (error) {
      if (error.name !== "AbortError")
        status.textContent =
          error.message ||
          "The example picture is unavailable. Please try again.";
    } finally {
      button.disabled = shown || controller.signal.aborted;
    }
  });
  scope.cleanup(() => controller.abort());
  return () => controller.abort();
}
