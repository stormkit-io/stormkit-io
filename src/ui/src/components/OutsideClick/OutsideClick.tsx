import React, { useRef, useEffect } from "react";

interface Props {
  handler: (arg0: any) => void;
  children: React.ReactNode;
}

export default function OutsideClick(props: Props) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const { handler } = props;

  useEffect(() => {
    function handleClickOutside(event: Event): void {
      if (
        event.target != null &&
        wrapperRef.current &&
        !wrapperRef.current.contains(event.target as Node)
      ) {
        handler(event);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [wrapperRef]);

  return <div ref={wrapperRef}>{props.children}</div>;
}
