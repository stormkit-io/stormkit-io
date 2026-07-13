import { useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
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

export default function TabConfigGeneral({
  environment: env,
  app,
  setRefreshToken,
}: Props) {
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isLoading, setLoading] = useState(false);

  if (!env) {
    return <></>;
  }

  return (
    <Card
      id="serverless"
      component="form"
      sx={{ mb: 2 }}
      error={error}
      success={success}
      onSubmit={e => {
        e.preventDefault();

        const build = prepareBuildObject(
          buildFormValues(env, e.target as HTMLFormElement)
        );

        updateEnvironment({
          app,
          envId: env.id!,
          payload: {
            apiFolder: build.apiFolder,
            apiPathPrefix: build.apiPathPrefix,
          },
          setError,
          setLoading,
          setSuccess,
          setRefreshToken,
        });
      }}
    >
      <CardHeader
        title="Serverless functions"
        subtitle="Configure your application's serverless settings."
      />

      <Box sx={{ mb: 4 }}>
        <TextField
          label="API folder"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.apiFolder || "/api"}
          name="build.apiFolder"
          fullWidth
          slotProps={{
            inputLabel: {
              shrink: true,
            },
          }}
          placeholder="/api"
          sx={{ mb: 4 }}
          helperText={`The path to the \`api\` folder where your serverless functions reside.`}
        />

        <TextField
          label="API path prefix"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.apiPathPrefix || "/api"}
          name="build.apiPathPrefix"
          fullWidth
          slotProps={{
            inputLabel: {
              shrink: true,
            },
          }}
          placeholder="/api"
          helperText={"The URL prefix for accessing your serverless functions."}
        />
      </Box>

      <CardFooter>
        <Button
          type="submit"
          variant="contained"
          color="secondary"
          loading={isLoading}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}
