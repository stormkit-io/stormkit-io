import { useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Switch from "@mui/material/Switch";
import FormControlLabel from "@mui/material/FormControlLabel";
import Typography from "@mui/material/Typography";
import { grey } from "@mui/material/colors";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import {
  updateEnvironment,
  buildFormValues,
  prepareBuildObject,
  validateRedirects,
} from "../actions";
import RedirectsPlaygroundModal from "./RedirectsPlaygroundModal";
import RedirectsEditor from "./Editor";

interface Props {
  app: App;
  environment: Environment;
  setRefreshToken: (v: number) => void;
}

const defaultRedirects = `[

]`;

export default function TabConfigRedirects({
  environment: env,
  app,
  setRefreshToken,
}: Props) {
  const hasRedirects = Boolean(env.build.redirects);
  const initialRedirects =
    JSON.stringify(env.build.redirects, null, 2) || defaultRedirects;

  const [markdown, setMarkdown] = useState(env.build.markdown === true);
  const [markdownConvert, setMarkdownConvert] = useState(
    env.build.markdownConvert === true,
  );
  const [showModal, setShowModal] = useState(false);
  const [showRedirects, setShowRedirects] = useState(hasRedirects);
  const [redirects, setRedirects] = useState(initialRedirects);

  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isLoading, setLoading] = useState(false);

  if (!env) {
    return <></>;
  }

  return (
    <Card
      id="redirects"
      component="form"
      sx={{ mb: 2 }}
      error={error}
      success={success}
      onSubmit={e => {
        e.preventDefault();

        const values = buildFormValues(env, e.target as HTMLFormElement, {
          "build.redirects": showRedirects ? redirects : undefined,
        });

        if (!validateRedirects(values["build.redirects"] || "", setError)) {
          setError("Invalid redirects format.");
          return;
        }

        const build = prepareBuildObject(values);

        updateEnvironment({
          app,
          envId: env.id!,
          payload: {
            redirects: build.redirects,
            redirectsFile: build.redirectsFile,
            errorFile: build.errorFile,
            markdown,
            markdownConvert: markdown && markdownConvert,
          },
          setError,
          setLoading,
          setSuccess,
          setRefreshToken,
        });
      }}
    >
      <CardHeader
        title="Redirects"
        subtitle="Configure redirects and path rewrites for your application."
        actions={
          <Button variant="text" onClick={() => setShowModal(true)}>
            Playground
          </Button>
        }
      />
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Redirects file location"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.redirectsFile || ""}
          name="build.redirectsFile"
          fullWidth
          InputLabelProps={{
            shrink: true,
          }}
          placeholder="./_redirects"
          helperText={"The path to the `redirects` file from the build root."}
        />
      </Box>
      <Box sx={{ mb: 4 }}>
        <TextField
          label="Custom error file"
          variant="filled"
          autoComplete="off"
          defaultValue={env?.build.errorFile || ""}
          name="build.errorFile"
          fullWidth
          InputLabelProps={{
            shrink: true,
          }}
          placeholder="/404.html"
          helperText={
            "The error file displayed when Stormkit does not find a page. Default is 404.html."
          }
        />
      </Box>
      <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
        <FormControlLabel
          sx={{ pl: 0, ml: 0 }}
          label="Markdown representations"
          control={
            <Switch
              name="build.markdown"
              color="secondary"
              checked={markdown}
              onChange={e => {
                setMarkdown(e.target.checked);
              }}
            />
          }
          labelPlacement="start"
        />
        <Typography sx={{ color: "text.secondary" }}>
          Turn on to serve any <Box component="code">.md</Box> file in your
          output as a second representation of the page next to it. A client
          that sends <Box component="code">Accept: text/markdown</Box> gets the
          markdown, a browser still gets the HTML. Useful for letting agents
          read your documentation without scraping the rendered page.
        </Typography>
        {markdown && (
          <Box
            sx={{
              mt: 2,
              pt: 2,
              borderTop: "1px solid",
              borderColor: "container.transparent",
            }}
          >
            <FormControlLabel
              sx={{ pl: 0, ml: 0 }}
              label="Convert pages without a .md file"
              control={
                <Switch
                  name="build.markdownConvert"
                  color="secondary"
                  checked={markdownConvert}
                  onChange={e => {
                    setMarkdownConvert(e.target.checked);
                  }}
                />
              }
              labelPlacement="start"
            />
            <Typography sx={{ color: "text.secondary" }}>
              When a page ships no <Box component="code">.md</Box> file, convert
              its HTML instead. A <Box component="code">.md</Box> file you
              publish yourself is always used first. Pages that are rendered in
              the browser have no content to convert and keep serving HTML.
            </Typography>
          </Box>
        )}
      </Box>
      <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
        <FormControlLabel
          sx={{ pl: 0, ml: 0 }}
          label="Overwrite redirects"
          control={
            <Switch
              color="secondary"
              checked={showRedirects}
              onChange={() => {
                setShowRedirects(!showRedirects);
              }}
            />
          }
          labelPlacement="start"
        />
        <Typography sx={{ color: "text.secondary" }}>
          Turn on to overwrite redirects. These redirects will apply immediately
          to all deployments and will take precedence over redirects file.
        </Typography>
      </Box>
      {showRedirects && (
        <Box
          sx={{
            mb: 4,
            p: 2,
            bgcolor: "container.paper",
            borderBottom: `1px solid ${grey[900]}`,
          }}
        >
          <Typography sx={{ mb: 2, color: "text.secondary" }}>
            Redirects
          </Typography>
          <RedirectsEditor
            value={redirects}
            onChange={setRedirects}
            docsLink="https://www.stormkit.io/docs/features/redirects-and-path-rewrites"
          />
        </Box>
      )}

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
      {showModal && (
        <RedirectsPlaygroundModal
          env={env}
          appId={app.id}
          onClose={() => {
            setShowModal(false);
          }}
        />
      )}
    </Card>
  );
}
