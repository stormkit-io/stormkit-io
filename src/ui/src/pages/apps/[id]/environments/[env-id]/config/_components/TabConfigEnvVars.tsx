import type { FormValues } from "../actions";
import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import VisibilityIcon from "@mui/icons-material/Visibility";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import KeyValue from "~/components/FormV2/KeyValue";
import {
  updateEnvironment,
  buildFormValues,
  prepareBuildObject,
  revealEnvVars,
} from "../actions";

interface Props {
  app: App;
  environment: Environment;
  setRefreshToken: (v: number) => void;
}

export default function TabConfigEnvVars({
  environment: env,
  app,
  setRefreshToken,
}: Props) {
  // With no pre-configured variables there is nothing to reveal, so editing is
  // unlocked from the start (no Reveal button, Save enabled).
  const hasVars = Object.keys(env.build.vars || {}).length > 0;

  const [reset, setReset] = useState<number>();
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isLoading, setLoading] = useState(false);
  const [isChanged, setIsChanged] = useState(false);
  const [revealed, setRevealed] = useState(!hasVars);
  const [revealing, setRevealing] = useState(false);

  // Values arrive masked. kvDefault seeds the rows and only changes on
  // reveal/reset; vars is the live edited map used when saving.
  const [kvDefault, setKvDefault] = useState(() => env.build.vars || {});
  const [vars, setVars] = useState(() => env.build.vars || {});

  const reveal = () => {
    setRevealing(true);
    setError("");
    setSuccess("");

    revealEnvVars({ envId: env.id! })
      .then(real => {
        setKvDefault(real);
        setVars(real);
        setReset(Date.now());
        setRevealed(true);
      })
      .catch(() => {
        setError(
          "Something went wrong while revealing the variables. Please try again."
        );
      })
      .finally(() => {
        setRevealing(false);
      });
  };

  return (
    <Card
      id="env-vars"
      component="form"
      sx={{ mb: 2 }}
      error={error}
      success={success}
      info={
        revealed
          ? undefined
          : "Values are hidden. Reveal them to view or edit — each reveal is recorded in the activity feed."
      }
      onSubmit={e => {
        e.preventDefault();

        const values: FormValues = buildFormValues(
          { ...env, build: { ...env.build, vars } },
          e.target as HTMLFormElement
        );

        updateEnvironment({
          app,
          envId: env.id!,
          payload: { envVars: prepareBuildObject(values).vars },
          setError,
          setLoading,
          setSuccess,
          setRefreshToken,
        }).then(() => {
          setIsChanged(false);
        });
      }}
    >
      <CardHeader
        title="Environment variables"
        subtitle="These variables will be available to build time, status checks and serverless runtime."
        actions={
          revealed ? undefined : (
            <Button
              type="button"
              variant="text"
              startIcon={<VisibilityIcon />}
              loading={revealing}
              onClick={reveal}
            >
              Reveal values
            </Button>
          )
        }
      />
      <Box sx={{ mb: 2 }}>
        <KeyValue
          resetToken={reset}
          inputName="build.vars"
          keyName="Name"
          valName="Value"
          keyPlaceholder="NODE_ENV"
          valPlaceholder="production"
          isSensitive
          disabled={!revealed}
          onChange={newVars => {
            setVars(newVars);
            setIsChanged(true);
          }}
          onModalOpen={() => {
            setSuccess("");
            setError("");
          }}
          defaultValue={kvDefault}
        />
      </Box>
      <CardFooter>
        <Button
          type="reset"
          variant="text"
          color="info"
          sx={{
            mr: 2,
            opacity: isChanged ? 1 : 0,
            visibility: isChanged ? "visible" : "hidden",
            transition: "all 0.35s ease-in",
          }}
          onClick={() => {
            const masked = env.build.vars || {};
            setKvDefault(masked);
            setVars(masked);
            setReset(Date.now());
            setIsChanged(false);
            setRevealed(!hasVars);
          }}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          variant="contained"
          color="secondary"
          loading={isLoading}
          disabled={!revealed}
          sx={{ textTransform: "capitalize" }}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}
