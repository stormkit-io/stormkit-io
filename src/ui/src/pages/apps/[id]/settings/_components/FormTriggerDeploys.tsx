import React, { FormEventHandler, useState } from "react";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import Card from "~/components/Card";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CopyBox from "~/components/CopyBox";
import api from "~/utils/api/Api";
import { deleteTrigger } from "../actions";
import type { AppSettings } from "../types.d";

interface Props {
  additionalSettings: AppSettings;
  app: App;
  environments: Array<Environment>;
  setAdditionalSettings: (value: React.SetStateAction<AppSettings>) => void;
}

const FormTriggerDeploys: React.FC<Props> = ({
  app,
  environments,
  additionalSettings,
  setAdditionalSettings,
}): React.ReactElement => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");

  const handleSubmit: FormEventHandler<HTMLElement> = e => {
    e.preventDefault();

    setLoading(true);
    setError("");

    api
      .put(`/app/deploy-trigger`, { appId: app.id })
      .then(() => {
        setLoading(false);
        window.location.reload();
      })
      .catch(() => {
        setLoading(false);
        setError(
          "Something went wrong while generating a new hash. Please try again and if the problem persists contact us on Discord or Email."
        );
      });
  };

  return (
    <Card component="form" error={error} onSubmit={handleSubmit} sx={{ mb: 4 }}>
      <CardHeader
        title="Deploy triggers"
        subtitle="Create endpoints to trigger deployments programmatically."
      />
      <Box sx={{ mb: 4 }}>
        {!additionalSettings.deployTrigger ? (
          <Alert color="info">
            <Typography>
              Click Generate button to create an endpoint for each environment
              to trigger deployments programmatically.{" "}
              <Link href="https://www.stormkit.io/docs/deployments#trigger">
                Learn more
              </Link>
              .
            </Typography>
          </Alert>
        ) : (
          <>
            {environments.map((env, i) => (
              <Box key={env.id} sx={{ mb: 4 }}>
                <Typography sx={{ mb: 1 }}>{env.name}</Typography>
                <CopyBox
                  value={`${api.baseurl}/hooks/app/${app.id}/deploy/${additionalSettings.deployTrigger}/${env.id}`}
                />
              </Box>
            ))}
            <Alert color="info">
              <Typography>
                You can send a <b>POST</b> or <b>GET</b> request to initiate a
                deployment for the given environment.{" "}
                <Link href="https://www.stormkit.io/docs/deployments#trigger">
                  Learn more
                </Link>
                .
              </Typography>
            </Alert>
          </>
        )}
      </Box>
      <CardFooter
        sx={
          additionalSettings.deployTrigger
            ? {
                display: "flex",
                justifyContent: "space-between",
              }
            : {}
        }
      >
        {additionalSettings.deployTrigger && (
          <Button
            color="info"
            variant="text"
            sx={{ mr: 2 }}
            loading={loading}
            onClick={e => {
              e.preventDefault();
              setLoading(true);

              setAdditionalSettings({
                ...additionalSettings,
                deployTrigger: undefined,
              });

              deleteTrigger(app.id)
                .then(() => {
                  setLoading(false);
                })
                .catch(e => {
                  setError(e.message);
                  setLoading(false);
                });
            }}
          >
            Delete endpoints
          </Button>
        )}
        <Button
          loading={loading}
          type="submit"
          color="secondary"
          variant="contained"
        >
          {additionalSettings.deployTrigger
            ? "Generate a new endpoint"
            : "Generate"}
        </Button>
      </CardFooter>
    </Card>
  );
};

export default FormTriggerDeploys;
