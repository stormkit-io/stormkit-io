import type { MaintenanceConfig } from "./actions";
import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import Switch from "@mui/material/Switch";
import Typography from "@mui/material/Typography";
import LensIcon from "@mui/icons-material/Lens";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { useFetchMaintenanceConfig, updateMaintenanceConfig } from "./actions";

interface Props {
  app: App;
  environment: Environment;
}

export default function Maintenance({ app, environment: env }: Props) {
  const [refreshToken, setRefreshToken] = useState<number>();
  const [formError, setFormError] = useState<string>();
  const [sendLoading, setSendLoading] = useState<boolean>(false);
  const [success, setSuccess] = useState<string>();
  const { loading, error, config } = useFetchMaintenanceConfig({
    appId: app.id,
    envId: env.id!,
    refreshToken,
  });

  const isActive = config === "on";

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    const maintenance = data.maintenance === "on" ? "on" : "";

    setSendLoading(true);

    updateMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      maintenance: maintenance as MaintenanceConfig,
    })
      .then(() => {
        setRefreshToken(Date.now());
        setSuccess("Maintenance mode configuration updated successfully.");
        setFormError(undefined);
      })
      .catch(async e => {
        const data = await e.json();

        setFormError(
          data.error ||
            "Something went wrong while updating maintenance mode configuration."
        );
      })
      .finally(() => {
        setSendLoading(false);
      });
  };

  return (
    <Card
      id="maintenance"
      component="form"
      loading={loading}
      error={error || formError}
      success={success}
      onSubmit={submitHandler}
      sx={{ mb: 4 }}
    >
      <CardHeader
        title="Maintenance mode"
        subtitle="Take your site offline temporarily and display a maintenance page to visitors."
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
      <Box sx={{ mb: 4 }}>
        <FormControlLabel
          control={
            <Switch
              name="maintenance"
              color="secondary"
              defaultChecked={isActive}
            />
          }
          label="Turn on maintenance mode"
        />
        <Typography sx={{ color: "text.secondary", fontSize: 12, mt: 1 }}>
          While turned on, public traffic receives a maintenance page with a
          503 status code. Deployments and settings remain untouched.
        </Typography>
      </Box>
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
