import React, { useContext, useState } from "react";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Box from "@mui/material/Box";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { setDomain } from "./actions";

interface Props {
  onClose: () => void;
  setRefreshToken: (val: number) => void;
}

const DomainModal: React.FC<Props> = ({ onClose, setRefreshToken }) => {
  const { app } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);

  return (
    <Modal open onClose={onClose}>
      <Card
        component="form"
        error={error}
        onSubmit={e => {
          e.preventDefault();
          const form = e.target as HTMLFormElement;
          const data = Object.fromEntries(
            new FormData(form).entries()
          ) as Record<string, string>;

          setLoading(true);
          setError(undefined);
          setDomain({
            appId: app.id,
            envId: environment.id!,
            values: { domain: data.domain },
          })
            .then(() => {
              setRefreshToken(Date.now());
              onClose?.();
            })
            .catch(async res => {
              let error = "Please provide a valid domain name.";

              try {
                const data = await res.json();
                error = data.error;
              } catch {}

              setError(
                res.status === 400
                  ? error
                  : res.status === 429
                  ? "You have issued too many requests. Please wait a while before retrying."
                  : "Something went wrong while setting up the domain. Make sure it is a valid domain."
              );
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader title="Setup a new domain" />
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Domain name"
            variant="filled"
            autoComplete="off"
            defaultValue={""}
            fullWidth
            name="domain"
            autoFocus
            required
            placeholder="e.g. www.stormkit.io"
            inputProps={{
              "aria-label": "Domain name",
            }}
          />
        </Box>
        <CardFooter>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            loading={loading}
          >
            Start verification process
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
};

export default DomainModal;
