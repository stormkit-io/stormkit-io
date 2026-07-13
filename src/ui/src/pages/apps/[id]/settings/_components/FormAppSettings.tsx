import type { AppSettings } from "../types.d";
import React, { useState, FormEventHandler } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { updateAdditionalSettings } from "../actions";
import { toRepoAddr } from "../helpers";

const devDomain = process.env.SK_DEV_DOMAIN || "stormkit.dev";

interface Props {
  app: App;
  additionalSettings: AppSettings;
}

const FormAppSettings: React.FC<Props> = ({ app, additionalSettings }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();

  const handleSubmit: FormEventHandler<HTMLElement> = e => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    setLoading(true);
    setError("");
    setSuccess("");

    updateAdditionalSettings({
      app,
      values: data,
    })
      .then(() => {
        setSuccess("Settings were saved successfully.");
      })
      .catch(async res => {
        const { errors } = await res.json();

        setError(
          Object.keys(errors)
            .map(k => errors[k])
            .join(", ")
        );
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Card
      component="form"
      error={error}
      success={success}
      onSubmit={handleSubmit}
      sx={{ mb: 4 }}
    >
      <CardHeader
        title="App settings"
        subtitle="General settings of your application."
      />
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Display name"
          name="displayName"
          variant="filled"
          required
          defaultValue={app.displayName}
          InputProps={{
            endAdornment: (
              <Typography sx={{ opacity: 0.5, fontSize: 12 }}>
                .{devDomain}
              </Typography>
            ),
          }}
          fullWidth
        />
        <Typography sx={{ px: 1.25, pt: 2, opacity: 0.5 }}>
          The display name has to be unique over the whole Stormkit application
          ecosystem. It will be used as a prefix for your deployment endpoints.
          You can change it as many times as you wish, however be aware that as
          soon as your display name is changed, the old one will be available
          for others to use.
        </Typography>
      </Box>
      {!app.isBare && (
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Repository"
            name="repo"
            variant="filled"
            required
            placeholder="e.g. https://github.com/stormkit-io/app-stormkit-io"
            defaultValue={toRepoAddr(app.repo)}
            fullWidth
          />
          <Typography sx={{ px: 1.25, pt: 2, opacity: 0.5 }}>
            The repo address of your application.
          </Typography>
        </Box>
      )}
      <CardFooter>
        <Button
          loading={loading}
          type="submit"
          variant="contained"
          color="secondary"
        >
          Update
        </Button>
      </CardFooter>
    </Card>
  );
};

export default FormAppSettings;
