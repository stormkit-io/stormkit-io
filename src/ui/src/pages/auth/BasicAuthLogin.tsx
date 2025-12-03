import { useState } from "react";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import api from "~/utils/api/Api";

export default function BasicAuthLogin() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();

  return (
    <Card
      component="form"
      sx={{ width: "100%", border: "none" }}
      error={error}
      onSubmit={e => {
        e.preventDefault();

        const form = e.target as HTMLFormElement;
        const data = Object.fromEntries(new FormData(form).entries()) as Record<
          string,
          string
        >;

        if (!data.email || !data.password) {
          setError("Email and password are required.");
          return;
        }

        setLoading(true);
        setError(undefined);

        api
          .post<{ sessionToken?: string; user?: User }>(
            "/auth/admin/login",
            data
          )
          .then(({ sessionToken }) => {
            if (sessionToken) {
              api.setAuthToken(sessionToken); // adds it to local storage
              window.location.reload();
            }
          })
          .catch(() => {
            setError("Invalid credentials.");
          })
          .finally(() => {
            setLoading(false);
          });
      }}
    >
      <CardHeader
        title="Authentication"
        subtitle="Login to your administrator account"
        sx={{ textAlign: "center" }}
      />
      <TextField
        variant="filled"
        name="email"
        label="Email"
        placeholder="Admin email address"
        fullWidth
        sx={{ mb: 4 }}
      />
      <TextField
        variant="filled"
        type="password"
        name="password"
        label="Password"
        placeholder="Admin password"
        fullWidth
        sx={{ mb: 4 }}
      />
      <Button
        variant="contained"
        color="secondary"
        type="submit"
        loading={loading}
        sx={{ mb: error ? 4 : 0 }}
        fullWidth
      >
        Login
      </Button>
    </Card>
  );
}
