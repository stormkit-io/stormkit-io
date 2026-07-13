import React, { forwardRef, ReactElement } from "react";
import cn from "classnames";
import { Link as OriginalLink } from "react-router-dom";

type Props = {
  to: string;
  secondary?: boolean;
  tertiary?: boolean;
} & React.HTMLProps<HTMLAnchorElement>;

const Link = forwardRef<HTMLAnchorElement, Props>(
  (props: Props, ref): ReactElement => {
    const { to, children, secondary, tertiary, className, ...rest } = props;

    let isExternal = false;
    let classes: string | undefined = className;

    if (secondary) {
      classes = cn(className, "text-pink-50", "hover:text-red-50");
    } else if (tertiary) {
      classes = cn(
        className,
        "text-primary",
        "font-bold",
        "hover:text-pink-50"
      );
    }

    if (to.indexOf("http") === 0 || to.indexOf("//") === 0) {
      rest.rel = "noreferrer noopener";
      rest.target = "_blank";
      isExternal = true;
    } else if (to.indexOf("mailto:") === 0) {
      isExternal = true;
    }

    if (isExternal) {
      return (
        <a href={to} {...rest} ref={ref} className={classes}>
          {children}
        </a>
      );
    }

    return (
      <OriginalLink to={to} {...rest} className={classes} ref={ref}>
        {children}
      </OriginalLink>
    );
  }
);

export default Link;
