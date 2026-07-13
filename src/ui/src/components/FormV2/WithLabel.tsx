import React from "react";
import { Tooltip } from "@mui/material";
import cn from "classnames";

type ExtendedProps = {
  tooltip?: React.ReactNode;
  children: React.ReactNode;
  label: React.ReactNode;
  className?: string[] | string | Record<string, unknown>;
};

const InputWithLabel: React.FC<ExtendedProps> = ({
  label,
  className,
  children,
  tooltip,
}) => {
  return (
    <label className={cn("flex p-4 form-label", className)}>
      <div className="bg-black text-xs flex items-center justify-between px-4 min-w-28 border-b border-solid border-gray-40 min-h-12">
        {label}
        {tooltip && (
          <Tooltip title={tooltip}>
            <span className="fas fa-question-circle" />
          </Tooltip>
        )}
      </div>
      <div className="bg-blue-10 flex items-center w-full border-b border-solid border-gray-40">
        {children}
      </div>
    </label>
  );
};

export default InputWithLabel;
