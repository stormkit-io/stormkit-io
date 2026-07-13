import React, { ReactElement, ReactNode, FC } from "react";
import cn from "classnames";

type Props = {
  children: ReactNode;
  className?: string;
};

const Description: FC<Props> = ({
  children,
  className,
}: Props): ReactElement => {
  return (
    <div className={cn("p-3 text-sm color-gray-55", className)}>{children}</div>
  );
};

export default Description;
