import { createContext } from "react";

type theme = "dark" | "light";

interface RootContextProps {
  mode: theme;
  details?: InstanceDetails;
  loading?: boolean;
  setMode: (v: theme) => void;
  setRefreshToken?: (token: number) => void;
}

export const RootContext = createContext<RootContextProps>({
  mode: "dark",
  setMode: () => {},
});
