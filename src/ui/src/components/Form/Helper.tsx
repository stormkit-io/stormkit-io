import React from "react";
import cn, { Argument } from "classnames";

interface Props {
  children: React.ReactNode;
  className?: Argument;
}

const Helper: React.FC<Props> = ({ children, className }) => {
  return (
    <div className={cn("pt-2 px-3 text-sm text-gray-400", className)}>
      {children}
    </div>
  );
};

export default Helper;
