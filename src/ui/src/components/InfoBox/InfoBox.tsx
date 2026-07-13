import React, { useEffect, useRef, useState } from "react";
import cn from "classnames";
import Button from "~/components/Button";
import "./InfoBox.css";

const SUCCESS = "success";
const WARNING = "warning";
const ERROR = "error";
const DEFAULT = "default";

const icons = {
  [SUCCESS]: "fas fa-check",
  [WARNING]: "fas fa-radiation",
  [ERROR]: "fas fa-exclamation-triangle",
  [DEFAULT]: "fas fa-info",
};

const colors = {
  [DEFAULT]: {
    text: "gray-80",
    bg: "black",
    border: "blue-30",
    icon: "blue-30",
  },
  [SUCCESS]: {
    text: "white",
    bg: "green-50",
    border: "green-50",
    icon: "",
  },
  [WARNING]: {
    text: "black font-light",
    bg: "yellow-10",
    border: "",
    icon: "yellow-60",
  },
  [ERROR]: {
    text: "white",
    bg: "red-50",
    border: "",
    icon: "",
  },
};

type StateClass =
  | typeof SUCCESS
  | typeof WARNING
  | typeof ERROR
  | typeof DEFAULT;

interface Props {
  children: React.ReactNode;
  className?: string;
  type?: StateClass;
  showIcon?: boolean;
  scrollIntoView?: boolean;
  toaster?: boolean;
  dismissable?: boolean;
  baseline?: boolean;
  onDismissed?: () => void;
}

const InfoBox: React.FC<Props> & {
  ERROR: typeof ERROR;
  SUCCESS: typeof SUCCESS;
  WARNING: typeof WARNING;
} = ({
  children,
  className,
  scrollIntoView,
  showIcon = true,
  type = DEFAULT,
  toaster,
  dismissable,
  baseline,
  onDismissed,
}): React.ReactElement => {
  const [isOpen, setIsOpen] = useState<boolean>(true);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (ref.current?.scrollIntoView && scrollIntoView) {
      ref.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [ref, scrollIntoView]);

  const classes: Array<string> = [
    "px-4",
    "py-2",
    "flex",
    baseline ? "items-baseline" : "items-center",
    "text-sm",
    `bg-${colors[type].bg}`,
    `text-${colors[type].text}`,
    colors[type].border && `border border-solid border-${colors[type].border}`,
  ];

  if (isOpen === false) {
    return <></>;
  }

  return (
    <div
      ref={ref}
      className={cn(classes, className, {
        shadow: toaster,
        "infobox-toaster": toaster,
      })}
    >
      {showIcon && (
        <span
          className={cn(
            "inline-flex flex-auto flex-grow-0 min-w-6 max-w-6 h-6 items-center justify-center mr-4"
          )}
        >
          <span
            className={cn(icons[type], "text-lg", {
              [`text-${colors[type].icon}`]: Boolean(colors[type].icon),
            })}
          />
        </span>
      )}
      <div className="flex-auto">{children}</div>
      {dismissable && (
        <Button
          className="ml-4"
          styled={false}
          onClick={() => {
            setIsOpen(false);
            onDismissed && onDismissed();
          }}
        >
          <span className="fas fa-times font-bold text-xl" />
        </Button>
      )}
    </div>
  );
};

InfoBox.ERROR = ERROR;
InfoBox.SUCCESS = SUCCESS;
InfoBox.WARNING = WARNING;

export default InfoBox;
