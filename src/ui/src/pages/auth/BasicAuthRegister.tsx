import { useState } from "react";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import api from "~/utils/api/Api";

export default function BasicAuthRegister() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();

  return (
    <Card
      component="form"
      error={error}
      sx={{ width: "100%" }}
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

        if (data.password !== data.password_confirmation) {
          setError("Passwords do not match.");
          return;
        }

        setLoading(true);
        setError(undefined);

        api
          .post<{ user: User; sessionToken: string }>("/auth/admin/register", {
            email: data.email,
            password: data.password,
          })
          .then(({ sessionToken }) => {
            if (sessionToken) {
              api.setAuthToken(sessionToken); // adds it to local storage
              window.location.reload();
            }
          })
          .catch(async res => {
            if (res.status === 400) {
              const { error } = (await res.json()) as { error: string };
              setError(error);
            } else {
              setError("Something went wrong, try again.");
            }
          })
          .finally(() => {
            setLoading(false);
          });
      }}
    >
      <CardHeader
        title="Welcome to Stormkit"
        subtitle="Create an administrator account with full access"
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
      <TextField
        variant="filled"
        type="password"
        name="password_confirmation"
        label="Password confirmation"
        placeholder="Confirm your password"
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
        Create account
      </Button>
    </Card>
  );
}
