import { Children, Fragment, cloneElement, isValidElement } from "react";

// Server-selected values initialize editable React form fields. Submitting the
// form continues to use the existing Go endpoint and its validation/redirects.
export function Select({ children, ...props }) {
  const selected = [];
  function options(nodes) {
    return Children.map(nodes, (child) => {
      if (!isValidElement(child)) return child;
      if (child.type === Fragment || child.type === "optgroup") {
        return cloneElement(child, {}, options(child.props.children));
      }
      if (child.type !== "option") return child;
      if (child.props.selected) selected.push(child.props.value);
      return cloneElement(child, { selected: undefined });
    });
  }
  const choices = options(children);
  const defaultValue = selected.length
    ? props.multiple
      ? selected
      : selected[0]
    : undefined;
  return (
    <select defaultValue={defaultValue} {...props}>
      {choices}
    </select>
  );
}

export function Textarea({ children, ...props }) {
  const content = (nodes) =>
    Children.toArray(nodes)
      .map((child) =>
        isValidElement(child) ? content(child.props.children) : child,
      )
      .join("");
  return <textarea defaultValue={content(children)} {...props} />;
}
