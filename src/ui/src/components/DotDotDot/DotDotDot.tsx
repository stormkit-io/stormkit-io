import React, { useState, cloneElement } from "react";
import cn from "classnames";
import OutsideClick from "~/components/OutsideClick";
import Button from "~/components/Button";

const DotDotDot: React.FC<React.HTMLAttributes<HTMLButtonElement>> & {
  Item: React.FC<ItemProps>;
} = ({ children, className, ...rest }): React.ReactElement => {
  const [isOpen, toggleVisibility] = useState(false);
  const childArray = React.Children.toArray(children);

  return (
    <OutsideClick handler={() => toggleVisibility(false)}>
      <div className={cn("relative", className)}>
        <Button
          styled={false}
          onClick={() => toggleVisibility(!isOpen)}
          {...rest}
        >
          <i className="fas fa-ellipsis-h" />
        </Button>
        {isOpen && (
          <div className="flex flex-col min-w-56 absolute right-0 rounded shadow bg-white z-50 items-start mt-4 text-left">
            {childArray.map((child: React.ReactNode, index: number) => {
              if (!React.isValidElement(child)) {
                throw new Error(
                  "[DotDotDot]: Invalid element provided as a child"
                );
              }

              return cloneElement(child, {
                toggleVisibility,
                isLast: childArray.length - 1 === index,
              });
            })}
          </div>
        )}
      </div>
    </OutsideClick>
  );
};

interface ItemProps {
  icon?: string;
  href?: string;
  isLast?: boolean;
  disabled?: boolean;
  className?: string;
  children: React.ReactNode;
  onClick?: (arg0: () => void) => void;
  toggleVisibility?: (arg0: boolean) => void;
}

const Item: React.FC<ItemProps> = ({
  icon,
  href,
  children,
  onClick,
  toggleVisibility,
  isLast,
  className,
  disabled,
  ...rest
}) => (
  <Button
    as="div"
    className={cn(
      "border-solid border-gray-80 p-4 w-full inline-flex items-center",
      {
        "hover:bg-gray-90": !disabled,
        "hover:text-pink-50": !disabled,
        "opacity-50": disabled,
        "cursor-not-allowed": disabled,
        "border-b": !isLast,
      },
      className
    )}
    disabled={disabled}
    href={href}
    styled={false}
    {...rest}
    onClick={e => {
      e.preventDefault();

      if (disabled) {
        return;
      }

      if (typeof onClick === "function") {
        toggleVisibility && onClick(() => toggleVisibility(false));
      }
    }}
  >
    {icon && <i className={icon} />}
    {children}
  </Button>
);

DotDotDot.Item = Item;

export default DotDotDot;
