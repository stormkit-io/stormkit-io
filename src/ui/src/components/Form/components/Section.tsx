import React, { ReactNode, FC, ReactElement } from "react";
import cn from "classnames";

interface Props {
  paddingTop?: string; // The tailwind class for the marginBottom
  marginBottom?: string; // The tailwind class for the marginBottom
  className?: string;
  children: ReactNode;
  label: ReactNode;
}

const Section: FC<Props> = ({
  children,
  className,
  label,
  paddingTop = "pt-4",
  marginBottom = "mb-8",
}: Props): ReactElement => {
  return (
    <div className={cn("flex", marginBottom, className)}>
      <div className={cn("flex-auto min-w-48 max-w-48 font-bold", paddingTop)}>
        {label}
      </div>
      <div className="flex-auto">{children}</div>
    </div>
  );
};

export default Section;
