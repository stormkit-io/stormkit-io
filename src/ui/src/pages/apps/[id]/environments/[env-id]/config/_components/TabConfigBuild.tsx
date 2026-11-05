import type { FormValues } from "../actions";
import { useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Switch from "@mui/material/Switch";
import Typography from "@mui/material/Typography";
import FormControlLabel from "@mui/material/FormControlLabel";
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
  const [root, setRoot] = useState(env?.build?.workDir || "./");
  const [skipUnchangedBuildRoot, setSkipUnchangedBuildRoot] = useState(
    env?.build?.skipUnchangedBuildRoot === true,
  );

  // A build root pointing at the repository root cannot filter anything out.
  const isRepoRoot = ["", ".", "./", "/"].includes(root.trim());

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

        // workDir and the path filter switch are controlled inputs (no form
        // name), so feed them in directly.
        const values: FormValues = buildFormValues(
          env,
          e.target as HTMLFormElement,
          {
            "build.workDir": root,
            "build.skipUnchangedBuildRoot": skipUnchangedBuildRoot
              ? "on"
              : "off",
          }
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
            skipUnchangedBuildRoot: build.skipUnchangedBuildRoot,
            watchPaths: build.watchPaths,
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
      <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
        <FormControlLabel
          sx={{ pl: 0, ml: 0 }}
          label="Deploy only when the build root changes"
          control={
            <Switch
              color="secondary"
              checked={skipUnchangedBuildRoot}
              onChange={e => {
                setSkipUnchangedBuildRoot(e.target.checked);
              }}
            />
          }
          labelPlacement="start"
        />
        <Typography sx={{ opacity: 0.5 }}>
          For monorepos. When turned on, a push only deploys this environment if
          it touches the build root, one of the watch paths below, or a file at
          the repository root. Available for GitHub and GitLab push events.
          {isRepoRoot &&
            " This environment builds from the repository root, so every change" +
              " already affects it and the filter has no effect."}
        </Typography>
      </Box>
      <Box sx={{ mb: 4, display: skipUnchangedBuildRoot ? "block" : "none" }}>
        <TextField
          label="Watch paths"
          variant="filled"
          autoComplete="off"
          multiline
          minRows={2}
          defaultValue={env?.build.watchPaths?.join("\n") || ""}
          fullWidth
          name="build.watchPaths"
          placeholder={"packages/ui\npackages/config"}
          helperText="One path per line, relative to the repository root. Add the shared workspace packages this environment depends on, so a change to them still triggers a deployment."
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
