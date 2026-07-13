import { useState } from "react";
import Button from "@mui/material/Button";
import TextInput from "@mui/material/TextField";
import Box from "@mui/material/Box";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { updateEnvironment } from "../../actions";
import { buildStatusChecks } from "./helpers";

interface Props {
  env: Environment;
  app: App;
  statusCheckIndex: number;
  setRefreshToken: (d: number) => void;
  onClose: () => void;
}

export default function StatusChecksModal({
  app,
  env,
  onClose,
  statusCheckIndex,
  setRefreshToken,
}: Props) {
  const statusCheck = env?.build?.statusChecks?.[statusCheckIndex];
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [name, setName] = useState(statusCheck?.name || "");
  const [cmd, setCmd] = useState(statusCheck?.cmd || "");
  const [desc, setDesc] = useState(statusCheck?.description || "");

  return (
    <Modal open onClose={onClose} maxWidth="md">
      <Card
        data-testid="status-checks-modal"
        component="form"
        error={error}
        success={success}
        onSubmit={e => {
          e.preventDefault();

          let checks: StatusCheck[];

          if (cmd.trim() == "") {
            setError("Command is a required field.");
            return;
          }

          try {
            checks = buildStatusChecks(
              env.build.statusChecks || [],
              { name, cmd, description: desc },
              statusCheckIndex
            );
          } catch {
            return setError(
              "Duplicate status check. You can run the same command only once."
            );
          }

          updateEnvironment({
            app,
            envId: env.id!,
            payload: { statusChecks: checks },
            setError,
            setLoading,
            setSuccess,
            setRefreshToken,
            successMsg: `Status check ${
              statusCheckIndex > -1 ? "created" : "updated"
            } successfully.`,
          }).then(onClose);
        }}
      >
        <CardHeader title="Status Check" />
        <Box sx={{ mb: 4 }}>
          <TextInput
            variant="filled"
            label="Command"
            placeholder="npm run e2e"
            fullWidth
            autoFocus
            autoComplete="off"
            value={cmd}
            onChange={e => setCmd(e.target.value)}
            helperText="The command to execute. Environment variables will be made available to this command."
          />
        </Box>
        <Box sx={{ mb: 4 }}>
          <TextInput
            variant="filled"
            label="Name"
            placeholder="End to end tests"
            fullWidth
            autoComplete="off"
            value={name}
            onChange={e => setName(e.target.value)}
            helperText={
              "The name that will be used to describe this status check."
            }
          />
        </Box>
        <Box sx={{ mb: 4 }}>
          <TextInput
            variant="filled"
            label="Description"
            placeholder="Run e2e tests after a deployment is complete."
            fullWidth
            autoComplete="off"
            value={desc}
            onChange={e => setDesc(e.target.value)}
            helperText={
              "The optional description to describe what this command does."
            }
          />
        </Box>
        <CardFooter>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={loading}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
