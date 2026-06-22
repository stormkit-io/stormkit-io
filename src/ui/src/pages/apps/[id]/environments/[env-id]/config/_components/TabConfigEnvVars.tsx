import type { FormValues } from "../actions";
import { useState, useMemo } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import KeyValue from "~/components/FormV2/KeyValue";
import {
  updateEnvironment,
  buildFormValues,
  prepareBuildObject,
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
  const [reset, setReset] = useState<number>();
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isLoading, setLoading] = useState(false);
  const [isChanged, setIsChanged] = useState(false);
  const [vars, setVars] = useState<Record<string, string>>(
    env.build.vars || {},
  );

  const defaultValue = useMemo(() => {
    return JSON.parse(
      JSON.stringify(
        env?.build?.vars || (env ? {} : { NODE_ENV: "development" }),
      ),
    );
  }, []);

  // Remove special variables
  delete defaultValue.SK_CWD;

  return (
    <Card
      id="env-vars"
      component="form"
      sx={{ mb: 2 }}
      error={error}
      success={success}
      onSubmit={e => {
        e.preventDefault();

        const values: FormValues = buildFormValues(
          { ...env, build: { ...env.build, vars } },
          e.target as HTMLFormElement,
        );

        const root = env.build?.vars?.["SK_CWD"];

        if (root) {
          values["build.vars"] = `${values["build.vars"]}\nSK_CWD=${root}`;
        }

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
          onChange={newVars => {
            setVars(newVars);
            setIsChanged(true);
          }}
          onModalOpen={() => {
            setSuccess("");
            setError("");
          }}
          defaultValue={defaultValue}
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
            setVars(env.build.vars || {});
            setReset(Date.now());
            setIsChanged(false);
          }}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          variant="contained"
          color="secondary"
          loading={isLoading}
          sx={{ textTransform: "capitalize" }}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}
