import { useMemo, useState } from "react";
import { Route, Routes, BrowserRouter } from "react-router-dom";
import { StyledEngineProvider, ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import Alert from "@mui/material/Alert";
import AuthContext from "./auth/Auth.context";
import { useFetchInstanceDetails } from "./auth/actions";
import routes from "./routes";
import createTheme from "./mui-theme";
import { RootContext } from "./Root.context";

interface Props {
  Router: typeof BrowserRouter;
}

const LS_KEY = "STORMKIT_MODE";

export default function Root({ Router }: Props) {
  const [refreshToken, setRefreshToken] = useState(0);
  const { details, loading, error } = useFetchInstanceDetails(refreshToken);

  const [mode, setMode] = useState<"dark" | "light">(
    (localStorage.getItem(LS_KEY) as "dark") || "dark"
  );

  const theme = useMemo(() => {
    return createTheme(mode);
  }, [mode]);

  return (
    <RootContext.Provider
      value={{
        details,
        loading,
        mode,
        setRefreshToken,
        setMode: v => {
          localStorage.setItem(LS_KEY, v);
          setMode(v);
        },
      }}
    >
      <StyledEngineProvider injectFirst>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          {error && (
            <Alert severity="error" sx={{ mb: 4 }}>
              {error}
            </Alert>
          )}
          <Router>
            <AuthContext>
              <Routes>
                {routes
                  .map(route => {
                    if (
                      route.cloudOnly &&
                      details?.stormkit?.edition !== "cloud"
                    ) {
                      return null;
                    }

                    return (
                      <Route
                        {...route}
                        key={
                          Array.isArray(route.path) ? route.path[0] : route.path
                        }
                      />
                    );
                  })
                  .filter(i => i)}
              </Routes>
            </AuthContext>
          </Router>
        </ThemeProvider>
      </StyledEngineProvider>
    </RootContext.Provider>
  );
}
