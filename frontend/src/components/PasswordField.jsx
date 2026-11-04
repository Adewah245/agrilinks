import { useState } from "react";

export default function PasswordField({
  id,
  name = id,
  label,
  icon = true,
  toggleId,
  ...props
}) {
  const [visible, setVisible] = useState(false);
  return (
    <>
      <label htmlFor={id}>{label}</label>
      <div className="password-wrapper">
        {icon && (
          <i className="fa-solid fa-lock input-icon" aria-hidden="true" />
        )}
        <input
          {...props}
          id={id}
          name={name}
          type={visible ? "text" : "password"}
        />
        <button
          type="button"
          id={toggleId}
          className="toggle-password"
          aria-controls={id}
          aria-label={`${visible ? "Hide" : "Show"} ${label.toLowerCase()}`}
          aria-pressed={visible}
          onClick={() => setVisible((value) => !value)}
        >
          <i
            className={`fa-solid ${visible ? "fa-eye-slash" : "fa-eye"}`}
            aria-hidden="true"
          />
        </button>
      </div>
    </>
  );
}
