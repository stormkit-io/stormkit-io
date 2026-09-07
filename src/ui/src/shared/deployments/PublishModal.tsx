import { useContext, useEffect, useRef, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import Modal from "~/components/Modal";
import CommitInfo from "./CommitInfo";
import { RootContext } from "~/pages/Root.context";
import { fetchPublishState, publishDeployments } from "./actions";
import AppChip from "./AppChip";

interface Props {
  deployment: DeploymentV2;
  onClose: () => void;
  onUpdate: () => void;

  // How often to ask whether the deployment is serving yet. Tests set this so
  // they do not have to spend real seconds waiting for a poll.
  pollInterval?: number;
}

// How often the deployment is asked whether it is serving yet.
const defaultPollInterval = 3000;

// Stormkit gives up warming a deployment after three minutes by default, and an
// instance can be configured to wait longer. The modal allows for that before
// it stops watching, rather than claiming an outcome it does not know.
const pollTimeout = 600000;

// A publish outlives a blip in the connection, so a handful of failed polls in
// a row is what it takes to stop watching.
const maxPollErrors = 5;

type Phase = "idle" | "publishing" | "published" | "failed" | "slow";

export default function PublishModal({
  deployment,
  onClose,
  onUpdate,
  pollInterval = defaultPollInterval,
}: Props) {
  const { mode } = useContext(RootContext);
  const [error, setError] = useState<string>();
  const [phase, setPhase] = useState<Phase>("idle");
  const isDark = mode === "dark";
  const startedAt = useRef(0);
  const sawWarmUp = useRef(false);

  useEffect(() => {
    if (phase !== "publishing") {
      return;
    }

    let unmounted = false;
    let errors = 0;

    const check = () => {
      fetchPublishState({ deploymentId: deployment.id })
        .then(({ published, isWarmingUp }) => {
          if (unmounted) {
            return;
          }

          if (published) {
            setPhase("published");
            onUpdate();
            return;
          }

          if (isWarmingUp) {
            sawWarmUp.current = true;
          }

          // The warm-up was under way and has stopped without the environment
          // moving: the deployment never answered.
          if (sawWarmUp.current && !isWarmingUp) {
            setPhase("failed");
            onUpdate();
            return;
          }

          if (Date.now() - startedAt.current > pollTimeout) {
            setPhase("slow");
            return;
          }

          errors = 0;
          timer = setTimeout(check, pollInterval);
        })
        .catch(() => {
          if (unmounted) {
            return;
          }

          errors += 1;

          if (errors >= maxPollErrors) {
            setPhase("slow");
            return;
          }

          timer = setTimeout(check, pollInterval);
        });
    };

    let timer = setTimeout(check, pollInterval);

    return () => {
      unmounted = true;
      clearTimeout(timer);
    };
  }, [phase, deployment.id, pollInterval]);

  // Warming up a deployment takes as long as it takes to boot, so the modal
  // says the wait is not something to sit through.
  const info =
    phase === "publishing"
      ? "Deployment is being warmed up... You can safely close this window, publishing carries on without you."
      : phase === "slow"
        ? "This is taking longer than usual. The publish carries on in the background — reopen this deployment later to see how it went."
        : undefined;

  return (
    <Modal
      open
      onClose={onClose}
      aria-labelledby={"Publish modal"}
      aria-describedby={"Publish deployment"}
    >
      <Card
        error={
          error ||
          (phase === "failed"
            ? "The deployment did not answer, so it was not published and your environment is still serving the previous one. Check the deployment's runtime logs to see why it did not start."
            : undefined)
        }
        success={
          phase === "published"
            ? "The environment is now serving this deployment."
            : undefined
        }
        info={info}
        contentPadding={false}
      >
        <CardHeader
          title="Publish deployment"
          subtitle="Stormkit starts the deployment first and moves traffic to it once it answers. Until then the current deployment keeps serving."
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
          {phase === "idle" || phase === "publishing" ? (
            <Button
              loading={phase === "publishing"}
              color="secondary"
              variant="contained"
              onClick={() => {
                setError(undefined);
                startedAt.current = Date.now();
                sawWarmUp.current = false;
                setPhase("publishing");

                publishDeployments({
                  percentages: { [deployment.id]: 100 },
                  envId: deployment.envId,
                  appId: deployment.appId,
                })
                  .then(() => {
                    onUpdate();
                  })
                  .catch(e => {
                    setPhase("idle");
                    setError(
                      typeof e === "string"
                        ? e
                        : "An error occurred while publishing. Please try again later.",
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
          ) : (
            <Button color="secondary" variant="contained" onClick={onClose}>
              Close
            </Button>
          )}
        </CardFooter>
      </Card>
    </Modal>
  );
}
