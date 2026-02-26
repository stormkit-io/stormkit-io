import type { AutoDeployValues } from "../actions";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Typography } from "@mui/material";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Switch from "@mui/material/Switch";
import Select from "@mui/material/Select";
import Option from "@mui/material/MenuItem";
import Button from "@mui/material/Button";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import InputLabel from "@mui/material/InputLabel";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import {
  useSubmitHandler,
  deleteEnvironment,
  computeAutoDeployValue,
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
  const navigate = useNavigate();
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [autoPublish, setAutoPublish] = useState(env.autoPublish || false);
  const [previewLinks, setPreviewLinks] = useState(env.build.previewLinks);
  const [autoDeploy, setAutoDeploy] = useState<AutoDeployValues>(
    computeAutoDeployValue(env),
  );

  const { submitHandler, error, isLoading, success } = useSubmitHandler({
    app,
    env,
    setRefreshToken,
    controlled: {
      autoPublish: autoPublish ? "on" : "off",
      "build.previewLinks": previewLinks ? "on" : "off",
    },
  });

  useEffect(() => {
    setAutoDeploy(computeAutoDeployValue(env));
  }, [env?.autoDeployBranches, env?.autoDeploy]);

  if (!env) {
    return <></>;
  }

  const isAutoDeployEnabled = autoDeploy !== "disabled";

  return (
    <Card
      id="general"
      component="form"
      error={error}
      success={success}
      sx={{ mb: 2 }}
      onSubmit={submitHandler}
    >
      <CardHeader
        title="General settings"
        subtitle="Use these settings to configure your environment details."
      />
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Environment name"
          name="name"
          fullWidth
          defaultValue={env.name}
          variant="filled"
          autoComplete="off"
        />
      </Box>
      {!app.isBare && (
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Branch"
            name="branch"
            fullWidth
            defaultValue={env.branch}
            variant="filled"
            autoComplete="off"
          />
        </Box>
      )}
      <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
        <FormControlLabel
          sx={{ pl: 0, ml: 0 }}
          label="Auto publish"
          control={
            <Switch
              name="autoPublish"
              color="secondary"
              checked={autoPublish}
              onChange={e => {
                setAutoPublish(e.target.checked);
              }}
            />
          }
          labelPlacement="start"
        />
        <Typography sx={{ opacity: 0.5 }}>
          Turn this feature on to automatically publish successful deployments
          on the default branch.
        </Typography>
      </Box>

      {!app.isBare && (
        <>
          <Box sx={{ mb: 4 }}>
            <FormControl variant="standard" fullWidth>
              <InputLabel id="auto-deploy-label" sx={{ pl: 2, pt: 1.25 }}>
                Auto deploy
              </InputLabel>
              <Select
                labelId="auto-deploy-label"
                name="autoDeploy"
                variant="filled"
                value={autoDeploy}
                fullWidth
                defaultValue="custom"
                onChange={e => {
                  setAutoDeploy(e.target.value as AutoDeployValues);
                }}
              >
                <Option value="disabled">Disabled</Option>
                <Option value="all">All branches</Option>
                <Option value="custom">Custom branches</Option>
                <Option value="custom_commit">Custom commits</Option>
              </Select>
            </FormControl>
          </Box>

          {autoDeploy === "custom" && (
            <Box sx={{ mb: 4 }}>
              <TextField
                variant="filled"
                name="autoDeployBranches"
                label="Auto deploy branches regexp"
                defaultValue={env?.autoDeployBranches || ""}
                InputProps={{
                  endAdornment: (
                    <Typography component="code" sx={{ ml: 1 }}>
                      /i
                    </Typography>
                  ),
                }}
                fullWidth
                helperText={
                  <>
                    <Typography
                      component="span"
                      sx={{ mb: 2, display: "block" }}
                    >
                      Specify which branches should be automatically deployed to
                      this environment. Below are some examples:
                    </Typography>
                    <Box component="span" sx={{ display: "block" }}>
                      <Box component="span" sx={{ mb: 2, display: "block" }}>
                        <Box component="code">^(?!dependabot).+</Box>{" "}
                        <Typography
                          component="span"
                          sx={{ mt: 0.5, display: "block" }}
                        >
                          Match anything that does not start with{" "}
                          <b>dependabot</b>
                        </Typography>
                      </Box>
                      <Box component="span" sx={{ display: "block" }}>
                        <Box component="code">^release-.+</Box>
                        <Typography
                          component="span"
                          sx={{ mt: 0.5, display: "block" }}
                        >
                          Match anything that starts with <b>release-</b>
                        </Typography>
                      </Box>
                    </Box>
                  </>
                }
              />
            </Box>
          )}

          {autoDeploy === "custom_commit" && (
            <Box sx={{ mb: 4 }}>
              <TextField
                variant="filled"
                name="autoDeployCommits"
                label="Auto deploy commit regexp"
                defaultValue={env?.autoDeployCommits || ""}
                InputProps={{
                  endAdornment: (
                    <Typography component="code" sx={{ ml: 1 }}>
                      /i
                    </Typography>
                  ),
                }}
                fullWidth
                helperText={
                  <>
                    <Typography
                      component="span"
                      sx={{ mb: 2, display: "block" }}
                    >
                      Specify which commits should be automatically deployed to
                      this environment. Below are some examples:
                    </Typography>
                    <Box component="span" sx={{ display: "block" }}>
                      <Box component="span" sx={{ mb: 2, display: "block" }}>
                        <Box component="code">^(?!release).+</Box>{" "}
                        <Typography
                          component="span"
                          sx={{ mt: 0.5, display: "block" }}
                        >
                          Match anything that does not start with <b>release</b>
                        </Typography>
                      </Box>
                      <Box component="span" sx={{ display: "block" }}>
                        <Box component="code">^chore\(release\):.+</Box>
                        <Typography
                          component="span"
                          sx={{ mt: 0.5, display: "block" }}
                        >
                          Match anything that starts with <b>chore(release):</b>
                        </Typography>
                      </Box>
                    </Box>
                  </>
                }
              />
            </Box>
          )}

          {isAutoDeployEnabled && (
            <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
              <FormControlLabel
                sx={{ pl: 0, ml: 0 }}
                label="Preview links"
                control={
                  <Switch
                    name="build.previewLinks"
                    color="secondary"
                    checked={
                      typeof previewLinks === "boolean" ? previewLinks : true
                    }
                    onChange={e => {
                      setPreviewLinks(e.target.checked);
                    }}
                  />
                }
                labelPlacement="start"
              />
              <Typography sx={{ opacity: 0.5 }}>
                Turn this feature on to leave preview links on the pull/merge
                request pages.
              </Typography>
            </Box>
          )}
        </>
      )}

      <CardFooter
        sx={{
          display: "flex",
          justifyContent: "space-between",
        }}
      >
        <Button
          type="button"
          variant="text"
          aria-label={`Delete ${env.name} environment`}
          onClick={() => {
            setIsDeleteModalOpen(true);
          }}
        >
          Delete {env.name} environment
        </Button>
        {isDeleteModalOpen && (
          <ConfirmModal
            onCancel={() => {
              setIsDeleteModalOpen(false);
            }}
            onConfirm={({ setLoading, setError }) => {
              setLoading(false);

              deleteEnvironment({ app, environment: env })
                .then(() => {
                  setLoading(false);
                  navigate(`/apps/${app.id}/environments`);
                  setRefreshToken?.(Date.now());
                })
                .catch(e => {
                  console.error(e);
                  setError(
                    "Something went wrong while deleting environment. Check the console.",
                  );
                });
            }}
          >
            This will completely delete the environment and associated
            deployments.
          </ConfirmModal>
        )}
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
