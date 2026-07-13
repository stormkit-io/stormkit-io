import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import Modal from "~/components/Modal";
import CommitInfo from "./CommitInfo";
import { RootContext } from "~/pages/Root.context";
import { publishDeployments } from "./actions";
import AppChip from "./AppChip";

interface Props {
  deployment: DeploymentV2;
  onClose: () => void;
  onUpdate: () => void;
}

export default function PublishModal({ deployment, onClose, onUpdate }: Props) {
  const { mode } = useContext(RootContext);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const isDark = mode === "dark";

  return (
    <Modal
      open
      onClose={onClose}
      aria-labelledby={"Publish modal"}
      aria-describedby={"Publish deployment"}
    >
      <Card error={error} contentPadding={false}>
        <CardHeader
          title="Publish deployment"
          subtitle="A published deployment will be promoted to the environment endpoint."
        />
        <Box>
          <CardRow key={deployment.id}>
            <CommitInfo
              deployment={deployment}
              showProject={false}
              clickable={false}
            />
          </CardRow>
        </Box>
        <CardFooter sx={{ textAlign: "center" }}>
          <Button
            loading={loading}
            color="secondary"
            variant="contained"
            onClick={() => {
              setLoading(true);
              publishDeployments({
                percentages: {
                  [deployment.id]: 100,
                },
                envId: deployment.envId,
                appId: deployment.appId,
              })
                .then(() => {
                  setLoading(false);
                  onUpdate();
                  onClose();
                })
                .catch(e => {
                  setLoading(false);
                  setError(
                    typeof e === "string"
                      ? e
                      : "An error occurred while publishing. Please try again later."
                  );
                });
            }}
          >
            <AppChip
              repo={deployment.repo}
              envName={deployment.envName}
              sx={{
                alignSelf: "flex-start",
                bgcolor: "transparent",
                color: isDark ? undefined : "white",
              }}
            >
              Publish to
            </AppChip>
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
