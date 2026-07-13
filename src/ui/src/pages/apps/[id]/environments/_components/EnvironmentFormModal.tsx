import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { insertEnvironment } from "../[env-id]/config/actions";

interface Props {
  onClose: () => void;
  isOpen: boolean;
  app: App;
}

interface FormValues {
  name: string;
  branch: string;
}

const EnvironmentFormModal: React.FC<Props> = ({ isOpen, app, onClose }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  return (
    <Modal open={isOpen} onClose={onClose}>
      <Card
        component="form"
        onSubmit={e => {
          e.preventDefault();

          const values = Object.fromEntries(
            new FormData(e.target as HTMLFormElement).entries()
          ) as unknown as FormValues;

          setError("");
          setLoading(true);

          insertEnvironment({
            app,
            values: { ...values, autoPublish: "on", autoDeploy: "all" },
          })
            .then(({ envId }) => {
              // We need to refresh a bunch of stuff, so it's easier to make a hard refresh.
              window.location.assign(`/apps/${app.id}/environments/${envId}`);
            })
            .catch(async res => {
              try {
                if (typeof res === "string") {
                  setError(res);
                } else {
                  const data = await res.json();

                  if (data.error) {
                    setError(data.error);
                  } else {
                    setError(Object.values(data.errors).join(""));
                  }
                }
              } catch {
                setError(
                  "Something went wrong while inserting the environment."
                );
              }
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title="Create new environment"
          subtitle="Environments allow you to work with different configurations."
        />
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Environment name"
            name="name"
            fullWidth
            defaultValue={""}
            variant="filled"
            autoComplete="off"
            placeholder="Development"
            required
            autoFocus
          />
        </Box>
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Environment branch"
            name="branch"
            fullWidth
            defaultValue={""}
            placeholder="release/development"
            variant="filled"
            autoComplete="off"
            helperText="The default branch for this environment."
            required
          />
        </Box>
        {error && (
          <Alert color="error" sx={{ mb: 4 }}>
            <AlertTitle>Error</AlertTitle>
            <Typography>{error}</Typography>
          </Alert>
        )}
        <CardFooter>
          <Button
            type="button"
            variant="text"
            color="info"
            sx={{ mr: 2 }}
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            loading={loading}
          >
            Create
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
};

export default EnvironmentFormModal;
