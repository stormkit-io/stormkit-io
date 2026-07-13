import React from "react";
import cn, { Argument } from "classnames";

interface Props {
  children: React.ReactNode;
  className?: Argument;
}

const Header: React.FC<Props> = ({ children, className }) => {
  return <h3 className={cn("font-bold", className)}>{children}</h3>;
};

export default Header;
