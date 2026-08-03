import type { FormValues } from "../actions";
import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import UpgradeButton from "~/components/UpgradeButton";
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
  const { user } = useContext(AuthContext);
  const { details } = useContext(RootContext);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isLoading, setLoading] = useState(false);
  const [root, setRoot] = useState(env?.build?.workDir || "./");

  // Build caching is available on self-hosted instances and for premium or
  // ultimate subscriptions on Stormkit Cloud.
  const isCacheAllowed =
    details?.stormkit?.edition !== "cloud" ||
    user?.package.id === "premium" ||
    user?.package.id === "ultimate";

  if (!env) {
    return <></>;
  }

  return (
    <Card
      id="build"
      component="form"
      sx={{ mb: 2 }}
      error={error}
      success={success}
      onSubmit={e => {
        e.preventDefault();

        // workDir is a controlled input (no form name), so feed it in directly.
        const values: FormValues = buildFormValues(
          env,
          e.target as HTMLFormElement,
          { "build.workDir": root }
        );

        const build = prepareBuildObject(values);

        updateEnvironment({
          app,
          envId: env.id!,
          payload: {
            installCmd: build.installCmd,
            buildCmd: build.buildCmd,
            distFolder: build.distFolder,
            workDir: build.workDir,
            cacheDirs: build.cacheDirs,
          },
          setError,
          setLoading,
          setSuccess,
          setRefreshToken,
        });
      }}
    >
      <CardHeader
        title="Build settings"
        subtitle="Use these settings to configure your build options."
      />
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Install command"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.installCmd || ""}
          fullWidth
          name="build.installCmd"
          placeholder="The command to install your dependencies"
          helperText={"Leave empty for auto-detection."}
        />
      </Box>
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Build command"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.buildCmd || ""}
          fullWidth
          name="build.buildCmd"
          placeholder="Defaults to 'npm run build' or 'yarn build' or 'pnpm build'"
          helperText={
            <>
              Concatenate multiple commands:{" "}
              <Box component="code" sx={{ fontSize: 11, px: 0.5, py: 0.25 }}>
                npm run test && npm run build
              </Box>
            </>
          }
        />
      </Box>
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Output folder"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.distFolder || "./"}
          fullWidth
          name="build.distFolder"
          placeholder="Defaults to `build`, `dist`, `output` or `.stormkit`"
          helperText={
            <>
              The folder containing your built assets. For many projects, this
              is either{" "}
              <Box component="code" sx={{ fontSize: 11, px: 0.5, py: 0.25 }}>
                dist
              </Box>{" "}
              <Box component="code" sx={{ fontSize: 11, px: 0.5, py: 0.25 }}>
                build
              </Box>{" "}
              <Box component="code" sx={{ fontSize: 11, px: 0.5, py: 0.25 }}>
                output
              </Box>
              .
            </>
          }
        />
      </Box>
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Build root"
          variant="filled"
          autoComplete="off"
          value={root}
          onChange={e => {
            setRoot(e.target.value);
          }}
          fullWidth
          placeholder="Defaults to `./`"
          helperText={"The working directory relative to the Repository root."}
        />
      </Box>
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Cache directories"
          variant="filled"
          autoComplete="off"
          multiline
          minRows={2}
          disabled={!isCacheAllowed}
          defaultValue={env?.build.cacheDirs?.join("\n") || ""}
          fullWidth
          name="build.cacheDirs"
          placeholder={".next/cache\n.turbo"}
          helperText={
            isCacheAllowed ? (
              "One directory per line, relative to the build root. Restored before the install step and saved after a successful build. Best for compiler caches like .next/cache, .turbo or node's build cache; caching node_modules helps incremental installs but not `npm ci`, which wipes it first. Leave empty to disable caching."
            ) : (
              <>
                Build caching speeds up deployments by reusing directories like{" "}
                <Box component="code" sx={{ fontSize: 11, px: 0.5, py: 0.25 }}>
                  .next/cache
                </Box>{" "}
                between builds. <UpgradeButton variant="text" /> to enable it.
              </>
            )
          }
        />
      </Box>

      <CardFooter>
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
