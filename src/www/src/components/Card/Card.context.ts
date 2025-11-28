import { createContext } from "react";

export type Size = "small" | "medium";

interface ContextProps {
  size?: Size;
}

export default createContext<ContextProps>({});
