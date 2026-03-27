import { useState, useContext, SubmitEventHandler } from "react";
import { html } from "@codemirror/lang-html";
import CodeMirror from "@uiw/react-codemirror";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Switch from "@mui/material/Switch";
import Select from "@mui/material/Select";
import Option from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import InputLabel from "@mui/material/InputLabel";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import DomainSelector from "~/shared/domains/DomainSelector";
import Modal from "~/components/Modal";
import { addSnippet, updateSnippet } from "./actions";

interface Props {
  snippet?: Snippet;
  closeModal: () => void;
  setRefreshToken: (t: number) => void;
}

interface FormValues {
  enabled: "on" | "off";
  title: string;
  content: string;
  path: string;
  injectLocation:
    | "head_append"
    | "body_append"
    | "head_prepend"
    | "body_prepend";
}

const defaultContent = "<script>\n    console.log('Hello world');\n</script>";

export default function SnippetModal({
  closeModal,
  snippet,
  setRefreshToken,
}: Props) {
  const { app } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const [error, setError] = useState<string | string[]>();
  const [loading, setLoading] = useState(false);
  const [selectedHosts, setSelectedHosts] = useState<string[]>();
  const [codeContent, setCodeContent] = useState<string>(
    snippet?.content || defaultContent,
  );

  const handleSubmit: SubmitEventHandler = e => {
    e.preventDefault();

    setLoading(true);
    setError(undefined);

    const handler = snippet?.id ? updateSnippet : addSnippet;
    const values = Object.fromEntries(
      new FormData(e.target as HTMLFormElement).entries(),
    ) as unknown as FormValues;

    const [location, prependOrAppend] = values.injectLocation.split("_");

    handler({
      appId: app.id,
      envId: environment.id!,
      snippet: {
        id: snippet?.id,
        title: values.title,
        content: values.content,
        enabled: values.enabled === "on",
        location: location === "head" ? "head" : "body",
        prepend: prependOrAppend === "prepend",
        rules: {
          path: values.path || snippet?.rules?.path,
          hosts: selectedHosts ? selectedHosts : snippet?.rules?.hosts,
        },
      },
    })
      .then(() => {
        setRefreshToken(Date.now());
        closeModal();
      })
      .catch(async e => {
        if (typeof e === "string") {
          setError(e);
        } else {
          const data = await e.json();

          if (typeof data.error === "string") {
            setError(data.error);
            return;
          }

          if (Array.isArray(data.errors) && data.errors.length > 0) {
            setError(data.errors);
            return;
          }

          setError("Something went wrong while saving snippet.");
        }
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Modal open onClose={closeModal}>
      <Card component="form" error={error} onSubmit={handleSubmit}>
        <CardHeader
          title={snippet?.id ? `Edit snippet #${snippet.id}` : "Create snippet"}
          subtitle={
            <>
              Snippets will be injected during response time into your document.
              <br />
              You can enable or disable any snippet without the need of a
              deployment.
            </>
          }
        />

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Title"
            name="title"
            fullWidth
            defaultValue={snippet?.title || ""}
            variant="filled"
            autoComplete="off"
            autoFocus
            helperText="The title will be used internally to distinguish between snippets."
            InputLabelProps={{
              shrink: true,
            }}
          />
        </Box>
        <Box sx={{ mb: 4 }}>
          <input name="content" type="hidden" value={codeContent} />
          <Box sx={{ bgcolor: "rgba(0,0,0,0.1)", fontSize: 12 }}>
            <CodeMirror
              maxHeight="200px"
              value={codeContent}
              extensions={[html()]}
              onChange={value => setCodeContent(value)}
              theme="dark"
            />
          </Box>
        </Box>
        <Box sx={{ mb: 4 }}>
          <FormControl variant="standard" fullWidth>
            <InputLabel id="inject-location" sx={{ pl: 2, pt: 1 }}>
              Where to inject
            </InputLabel>
            <Select
              labelId="inject-location"
              name="injectLocation"
              variant="filled"
              fullWidth
              defaultValue={`${snippet?.location || "head"}_${
                snippet?.prepend ? "prepend" : "append"
              }`}
            >
              <Option value="head_prepend">
                Prepend to Head {"(inserted right after <head>)"}
              </Option>
              <Option value="head_append">
                Append to Head {"(inserted right before </head>)"}
              </Option>
              <Option value="body_prepend">
                Prepend to Body {"(inserted right after <body>)"}
              </Option>
              <Option value="body_append">
                Append to Body {"(inserted right before </body>)"}
              </Option>
            </Select>
          </FormControl>
        </Box>
        <Box sx={{ mb: 4 }}>
          <DomainSelector
            variant="filled"
            label="Inject only for specified domains"
            appId={app.id}
            envId={environment.id!}
            selected={snippet?.rules?.hosts}
            onDomainSelect={value => {
              setSelectedHosts(value as string[]);
            }}
            multiple
            withDevDomains
            fullWidth
          />
        </Box>
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Inject only for specified paths"
            name="path"
            fullWidth
            defaultValue={snippet?.rules?.path || ""}
            variant="filled"
            autoComplete="off"
            helperText="Accepts POSIX regular expressions (e.g. ^(b|c)d.*"
            InputLabelProps={{
              shrink: true,
            }}
          />
        </Box>
        <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 2 }}>
          <FormControlLabel
            sx={{ pl: 0, ml: 0 }}
            label="Enabled"
            control={
              <Switch
                name="enabled"
                color="secondary"
                defaultChecked={snippet?.enabled || false}
              />
            }
            labelPlacement="start"
          />
          <Typography sx={{ color: "rgba(255, 255, 255, 0.7)", fontSize: 11 }}>
            Enable this snippet to inject it dynamically into your page.
          </Typography>
        </Box>
        <CardFooter>
          <Button type="button" onClick={closeModal}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            sx={{ ml: 2 }}
            loading={loading}
          >
            {snippet ? "Update" : "Create"}
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
