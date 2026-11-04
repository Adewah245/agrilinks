import {
  Children,
  Fragment,
  cloneElement,
  isValidElement,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";

function linksIn(children) {
  return Children.toArray(children).flatMap((child) =>
    !isValidElement(child)
      ? []
      : child.type === "a"
        ? [child.props.href]
        : linksIn(child.props.children),
  );
}

export default function Navbar({
  children,
  className = "",
  as: Tag = "header",
  ...props
}) {
  const [open, setOpen] = useState(false);
  const root = useRef(null);
  const toggle = useRef(null);
  const id = useId();

  useEffect(() => {
    const desktop = window.matchMedia("(min-width: 1101px)");
    const close = () => setOpen(false);
    const outside = (event) => {
      if (!root.current?.contains(event.target)) close();
    };
    const escape = (event) => {
      if (
        event.key === "Escape" &&
        root.current?.classList.contains("nav-open")
      ) {
        close();
        toggle.current?.focus();
      }
    };
    const resize = () =>
      document.documentElement.style.setProperty(
        "--navbar-height",
        `${root.current.getBoundingClientRect().height}px`,
      );
    const observer =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(resize);
    observer?.observe(root.current);
    resize();
    desktop.addEventListener("change", close);
    document.addEventListener("click", outside);
    document.addEventListener("keydown", escape);
    return () => {
      observer?.disconnect();
      desktop.removeEventListener("change", close);
      document.removeEventListener("click", outside);
      document.removeEventListener("keydown", escape);
    };
  }, []);

  function navigation(nodes) {
    return Children.map(nodes, (child) => {
      if (!isValidElement(child)) return child;
      if (child.props.className?.split(/\s+/).includes("nav-toggle"))
        return null;
      if (child.type !== "nav")
        return child.props.children
          ? cloneElement(child, {}, navigation(child.props.children))
          : child;
      const links = linksIn(child.props.children);
      return (
        <Fragment>
          <div className="nav-quick-actions">
            {links.includes("/cart") && (
              <a href="/cart" aria-label="Cart" title="Cart">
                🛒
              </a>
            )}
            {links.includes("/notifications") && (
              <a
                href="/notifications"
                aria-label="Notifications"
                title="Notifications"
              >
                🔔
              </a>
            )}
          </div>
          <button
            ref={toggle}
            className="nav-toggle"
            type="button"
            aria-controls={id}
            aria-expanded={open}
            aria-label={open ? "Close navigation" : "Open navigation"}
            onClick={() => setOpen((value) => !value)}
          >
            <span aria-hidden="true">{open ? "×" : "☰"}</span>
          </button>
          {cloneElement(child, { id, onClick: () => setOpen(false) })}
        </Fragment>
      );
    });
  }
  return (
    <Tag
      {...props}
      ref={root}
      className={`${className} nav-enhanced${open ? " nav-open" : ""}`}
    >
      {navigation(children)}
    </Tag>
  );
}
