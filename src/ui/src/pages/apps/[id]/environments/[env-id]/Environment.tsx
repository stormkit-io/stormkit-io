import { Routes, Route } from "react-router";
import Box from "@mui/material/Box";
import EnvironmentHeader from "./_components/EnvironmentHeader";
import routes from "./routes";
import EnvironmentContextProvider from "../Environment.context";

export default function Environment() {
  return (
    <EnvironmentContextProvider>
      <Box sx={{ display: "flex", flexDirection: "column", width: "100%" }}>
        <EnvironmentHeader />
        <Routes>
          {routes.map(route => (
            <Route {...route} path={route.path} key={route.path} />
          ))}
        </Routes>
      </Box>
    </EnvironmentContextProvider>
  );
}
