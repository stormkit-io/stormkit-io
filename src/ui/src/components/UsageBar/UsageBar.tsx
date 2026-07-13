import React from "react";
import cn from "classnames";

interface Props {
  used: number;
  total: number;
  className?: string;
}

const UsageBar: React.FC<Props> = ({
  used,
  total,
  className,
}): React.ReactElement => {
  const percentage = (used / total) * 100;

  return (
    <div className={cn("h-3 rounded overflow-hidden bg-gray-200", className)}>
      <div
        style={{ width: `${percentage}%` }}
        className="bg-pink-50 h-full"
      ></div>
    </div>
  );
};

export default UsageBar;
