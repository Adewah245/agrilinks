import { Children, cloneElement, isValidElement } from "react";
import Navbar from "./Navbar.jsx";

const paths = {
  home: ["M3 10 12 3l9 7", "M5 9v12h14V9", "M9 21v-8h6v8"],
  market: [
    "M3 10 5 4h14l2 6",
    "M5 13v8h14v-8",
    "M3 10c0 4 4 4 4 0 0 4 5 4 5 0 0 4 5 4 5 0 0 4 4 4 4 0",
  ],
  orders: ["M5 7h14l1 14H4L5 7Z", "M8 7V6a4 4 0 0 1 8 0v1"],
  chat: ["M21 11a8 8 0 0 1-8 8H6l-4 3V11a9 9 0 0 1 19 0Z", "M7 10h10M7 14h6"],
  learning: [
    "M12 5c-4-3-8-2-10-1v15c3-1 7-1 10 1 3-2 7-2 10-1V4c-2-1-6-2-10 1Z",
    "M12 5v15",
  ],
  assistant: [
    "M12 3v3",
    "M5 6h14v14H5Z",
    "M8 11h.01M16 11h.01M8 16h8",
    "M2 10v6M22 10v6",
  ],
  wallet: [
    "M20 8V5H5a2 2 0 0 0 0 4h15v12H5a2 2 0 0 1-2-2V7",
    "M20 12h-6v5h6",
    "M16 14.5h.01",
  ],
  delivery: [
    "M3 5h11v13H3Z",
    "M14 9h4l4 5v4h-8",
    "M6 18a2 2 0 1 0 4 0M16 18a2 2 0 1 0 4 0",
  ],
  profile: ["M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z", "M4 21v-2a8 8 0 0 1 16 0v2"],
  storage: ["M3 9 12 3l9 6v12H3Z", "M8 21V11h8v10M8 15h8"],
  chart: ["M4 3v18h17", "M8 16v-5M13 16V7M18 16V4"],
  shield: ["M12 3 4 6v6c0 4 3.5 7 8 9 4.5-2 8-5 8-9V6Z", "m8 12 3 3 5-6"],
  bell: ["M5 17h14l-2-3V9A5 5 0 0 0 7 9v5l-2 3Z", "M10 21h4"],
  exit: ["M10 4H4v16h6", "M9 12h12m-4-4 4 4-4 4"],
};
const iconGroups = [
  ["exit", ["logout"]],
  ["bell", ["notification"]],
  ["shield", ["security", "trust", "protection"]],
  ["chart", ["revenue", "insights", "sales"]],
  ["wallet", ["finance", "wallet", "payment"]],
  ["learning", ["learning", "learn"]],
  ["assistant", ["assistant"]],
  ["delivery", ["delivery", "deliveries"]],
  ["profile", ["profile", "users"]],
  ["chat", ["chat"]],
  ["storage", ["storage"]],
  ["market", ["market"]],
  ["orders", ["order", "cart"]],
];
const iconFor = (href = "") =>
  iconGroups.find(([, words]) =>
    words.some((word) => href.includes(word)),
  )?.[0] || "home";

export function WorkspaceIcon({ name = "home" }) {
  return (
    <svg
      className="workspace-icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.65"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {(paths[name] || paths.home).map((d, i) => (
        <path d={d} key={i} />
      ))}
    </svg>
  );
}

function decorate(nodes, current) {
  return Children.map(nodes, (child) => {
    if (!isValidElement(child)) return child;
    if (child.type === "a")
      return cloneElement(
        child,
        {
          "aria-current":
            child.props.href === current ? "page" : child.props["aria-current"],
        },
        <>
          <WorkspaceIcon name={iconFor(child.props.href)} />
          <span className="workspace-nav-text">{child.props.children}</span>
        </>,
      );
    return child.props.children
      ? cloneElement(child, {}, decorate(child.props.children, current))
      : child;
  });
}

function containsLink(nodes, href) {
  return Children.toArray(nodes).some(
    (child) =>
      isValidElement(child) &&
      ((child.type === "a" && child.props.href === href) ||
        containsLink(child.props.children, href)),
  );
}

export function WorkspaceSidebar({
  current = "/dashboard",
  staff = false,
  children,
}) {
  const links = staff
    ? [
        ["/admin", "Overview"],
        ["/admin/finance", "Finance and payments"],
        ["/admin/revenue", "Revenue and reports"],
        ["/wallet/admin", "Wallet requests"],
        ["/deliveries", "Deliveries"],
        ["/trust", "Trust badges"],
        ["/admin/security", "Account security"],
      ]
    : [
        ["/dashboard", "Overview"],
        ["/marketplace", "Marketplace"],
        ["/orders", "My orders"],
        ["/chat", "Messages"],
        ["/learning", "Learn farming"],
        ["/ai-assistant", "Farming assistant"],
        ["/wallet", "My wallet"],
        ["/deliveries", "Deliveries"],
        ["/profile", "My profile"],
      ];
  return (
    <Navbar
      as="aside"
      className="navbar workspace-sidebar"
      aria-label="Workspace navigation"
    >
      <a
        className="workspace-brand logo"
        href={staff ? "/admin" : "/dashboard"}
      >
        <img
          src="/static/assets/logo/agrilinks-nigeria-logo.svg"
          alt="Agrilinks Nigeria"
        />
        <span>{staff ? "STAFF WORKSPACE" : "YOUR WORKSPACE"}</span>
      </a>
      <nav aria-label="Main navigation">
        {children
          ? decorate(children, current)
          : links.map(([href, label]) => (
              <a
                key={href}
                href={href}
                aria-current={
                  current === href ||
                  (href === "/learning" && current.startsWith("/learn")) ||
                  (href === "/deliveries" && current === "/delivery-partner")
                    ? "page"
                    : undefined
                }
              >
                <WorkspaceIcon name={iconFor(href)} />
                <span className="workspace-nav-text">{label}</span>
              </a>
            ))}
        {!children && (
          <a
            className="workspace-mobile-exit"
            href={staff ? "/dashboard" : "/logout"}
          >
            <WorkspaceIcon name={staff ? "home" : "exit"} />
            <span>{staff ? "Back to my account" : "Log out"}</span>
          </a>
        )}
      </nav>
      <div className="workspace-sidebar-footer">
        <a className="workspace-learning-link" href="/learning">
          <WorkspaceIcon name="learning" />
          <strong>A little learning, every day</strong>
          <span>Explore our farming guides →</span>
        </a>
        {!containsLink(children, staff ? "/dashboard" : "/logout") && (
          <a className="workspace-exit" href={staff ? "/dashboard" : "/logout"}>
            <WorkspaceIcon name={staff ? "home" : "exit"} />
            {staff ? "Back to my account" : "Log out"}
          </a>
        )}
      </div>
    </Navbar>
  );
}

export function WorkspaceHeading({
  eyebrow = "YOUR WORKSPACE",
  title,
  description,
  children,
}) {
  return (
    <header className="workspace-heading">
      <div>
        <p className="workspace-eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        {description && <p className="workspace-description">{description}</p>}
      </div>
      {children && <div className="workspace-heading-actions">{children}</div>}
    </header>
  );
}

export default function Workspace({ current, staff, children }) {
  return (
    <div className="workspace-frame">
      <WorkspaceSidebar current={current} staff={staff} />
      <div className="workspace-content">{children}</div>
    </div>
  );
}
