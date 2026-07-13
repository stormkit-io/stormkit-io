import React from "react";
import cn from "classnames";
import Box from "@mui/material/Box";

interface Props {
  bg?: "bg-blue-50";
  maxWidth?: "max-w-2xl" | "max-w-none";
  margin?: "m-auto";
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string | string[] | Record<string, boolean>;
  children: React.ReactNode;
}

const Container: React.FC<Props> = ({
  title,
  subtitle,
  actions,
  children,
  className,
  maxWidth = "max-w-2xl",
}) => {
  return (
    <Box
      sx={{ width: "100%", bgcolor: "container.paper" }}
      className={cn(["w-full text-gray-80", maxWidth, className])}
    >
      {(title || actions) && (
        <div className="flex p-4">
          {(title || subtitle) && (
            <div className="flex flex-col">
              {title && (
                <h3 className="flex items-center font-bold">{title}</h3>
              )}
              {subtitle && <h4 className="flex items-center">{subtitle}</h4>}
            </div>
          )}
          {actions && (
            <div className="flex flex-1 justify-end text-sm ml-4">
              {actions}
            </div>
          )}
        </div>
      )}
      {children}
    </Box>
  );
};

export default Container;
