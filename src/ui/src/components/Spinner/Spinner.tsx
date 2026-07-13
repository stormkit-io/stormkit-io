import React from "react";
import cn from "classnames";
import "./Spinner.css";

interface Props {
  width?: number;
  height?: number;
  pageCenter?: boolean; // Whether to center in the middle of the page or not.
  primary?: boolean;
  secondary?: boolean;
  className?: string;
}

const Spinner: React.FC<Props> = ({
  width = 10,
  height = 10,
  pageCenter,
  primary,
  secondary,
  className,
}: Props): React.ReactElement => (
  <div
    className={cn(
      "spinner",
      "rounded-full",
      `w-${width}`,
      `h-${height}`,
      {
        "page-center": pageCenter,
        "bg-pink-50": primary,
        "bg-blue-20": secondary,
      },
      className
    )}
  >
    <div className="relative w-full h-full">
      <div className="spinner-bounce w-full h-full rounded-full bg-white absolute top-0 left-0" />
      <div className="spinner-bounce w-full h-full rounded-full bg-white absolute top-0 left-0" />
    </div>
  </div>
);

export default Spinner;
