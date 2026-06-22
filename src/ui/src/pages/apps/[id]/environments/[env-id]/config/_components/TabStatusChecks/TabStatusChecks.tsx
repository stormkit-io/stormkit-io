import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import StatusChecksModal from "./StatusCheckModal";
import { updateEnvironment } from "../../actions";
import { buildStatusChecks } from "./helpers";

interface Props {
  app: App;
  environment: Environment;
  setRefreshToken: (v: number) => void;
}

export default function TabConfigGeneral({
  environment: env,
  app,
  setRefreshToken,
}: Props) {
  const [showModal, toggleModal] = useState(false);
  const [modified, setModified] = useState<number>();
  const [deleted, setDeleted] = useState<number>();

  if (!env) {
    return <></>;
  }

  const statusChecks = env.build.statusChecks || [];

  return (
    <Card
      id="status-checks"
      component="form"
      sx={{ mb: 2 }}
      contentPadding={false}
      info={
        statusChecks.length === 0
          ? "You do not have any status checks for this environment."
          : ""
      }
    >
      <CardHeader
        title="Status Checks"
        subtitle="Set up post-deployment commands to validate your deployment."
      />
      {statusChecks.map((check, index) => (
        <CardRow
          key={check.cmd + check.name}
          menuItems={[
            {
              text: "Modify",
              onClick() {
                setModified(index);
                toggleModal(true);
              },
            },
            {
              text: "Delete",
              onClick() {
                setDeleted(index);
                toggleModal(false);
              },
            },
          ]}
        >
          <Box sx={{ display: "flex" }}>
            <Typography sx={{ mr: 1 }}>{check.cmd}</Typography>
            <Typography color="text.secondary">{check.name}</Typography>
          </Box>
        </CardRow>
      ))}

      <CardFooter>
        <Button
          type="button"
          variant="contained"
          color="secondary"
          onClick={() => {
            setModified(undefined);
            setDeleted(undefined);
            toggleModal(true);
          }}
        >
          Add status check
        </Button>
      </CardFooter>
      {showModal && (
        <StatusChecksModal
          app={app}
          env={env}
          setRefreshToken={setRefreshToken}
          statusCheckIndex={typeof modified === "undefined" ? -1 : modified}
          onClose={() => {
            setModified(undefined);
            setDeleted(undefined);
            toggleModal(false);
          }}
        />
      )}
      {typeof deleted !== "undefined" && (
        <ConfirmModal
          onCancel={() => {
            setDeleted(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            updateEnvironment({
              app,
              envId: env.id!,
              payload: {
                statusChecks: buildStatusChecks(
                  env.build.statusChecks || [],
                  undefined,
                  deleted
                ),
              },
              setError,
              setLoading,
              setSuccess: () => {},
              setRefreshToken,
            }).then(() => {
              setDeleted(undefined);
            });
          }}
        />
      )}
    </Card>
  );
}
