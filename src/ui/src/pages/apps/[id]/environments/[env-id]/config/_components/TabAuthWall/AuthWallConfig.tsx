import type { AuthWallConfig } from "./actions";
import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";
import LensIcon from "@mui/icons-material/Lens";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { useFetchAuthWallConfig, updateAuthWallConfig } from "./actions";

interface Props {
  app: App;
  environment: Environment;
}

export default function AuthWallConfigForm({ app, environment: env }: Props) {
  const [refreshToken, setRefreshToken] = useState<number>();
  const [formError, setFormError] = useState<string>();
  const [sendLoading, setSendLoading] = useState<boolean>(false);
  const [success, setSuccess] = useState<string>();
  const { loading, error, config } = useFetchAuthWallConfig({
    appId: app.id,
    envId: env.id!,
    refreshToken,
  });

  const isActive = config !== "";

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    if (data.authwall === "off") {
      data.authwall = "";
    }

    if (["dev", "all", ""].indexOf(data.authwall) === -1) {
      return setFormError("Invalid auth wall value");
    }

    setSendLoading(true);

    updateAuthWallConfig({
      appId: app.id,
      envId: env.id!,
      authwall: data.authwall as AuthWallConfig,
    })
      .then(() => {
        setRefreshToken(Date.now());
        setSuccess("Auth wall configuration updated successfully.");
        setFormError(undefined);
      })
      .catch(async e => {
        const data = await e.json();

        setFormError(
          data.error ||
            "Something went wrong while updating auth wall configuration."
        );
      })
      .finally(() => {
        setSendLoading(false);
      });
  };

  return (
    <Card
      id="auth-wall"
      component="form"
      loading={loading}
      error={error || formError}
      success={success}
      onSubmit={submitHandler}
      sx={{ mb: 4 }}
    >
      <CardHeader
        title="Auth wall"
        subtitle="Limit access to your deployments with an authentication wall."
        actions={
          <Box sx={{ alignSelf: "flex-start" }}>
            <LensIcon
              color={isActive ? "success" : "error"}
              sx={{ width: 12 }}
            />
            <Typography component="span" sx={{ ml: 1, fontSize: 12 }}>
              {isActive ? "Enabled" : "Disabled"}
            </Typography>
          </Box>
        }
      />
      <FormControl variant="standard" fullWidth sx={{ mb: 4 }}>
        <InputLabel id="app-runtime-settings" sx={{ pl: 2, pt: 1 }}>
          Status
        </InputLabel>
        <Select
          labelId="app-runtime-settings"
          variant="filled"
          name="authwall"
          defaultValue={config === "" || !config ? "off" : config}
          sx={{ minWidth: 250 }}
        >
          <MenuItem value={"off"}>Auth Wall is disabled</MenuItem>
          <MenuItem value={"all"}>
            Protect all endpoints including your domains
          </MenuItem>
          <MenuItem value={"dev"}>Protect only deployment previews</MenuItem>
        </Select>
      </FormControl>
      <CardFooter>
        <Button
          type="submit"
          variant="contained"
          color="secondary"
          loading={sendLoading}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}
