import { useState, useContext, FormEventHandler, useMemo } from "react";
import { json } from "@codemirror/lang-json";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Select from "@mui/material/Select";
import Option from "@mui/material/MenuItem";
import Button from "@mui/material/Button";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import InputLabel from "@mui/material/InputLabel";
import Switch from "@mui/material/Switch";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import Modal from "~/components/Modal";
import Form from "~/components/FormV2";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import KeyValue from "~/components/FormV2/KeyValue";
import CardFooter from "~/components/CardFooter";
import { RootContext } from "~/pages/Root.context";
import { upsertFunctionTrigger } from "./actions";
import DomainSelector from "~/shared/domains/DomainSelector";

interface Props {
  triggerFunction?: FunctionTrigger;
  app: App;
  environment: Environment;
  closeModal: () => void;
  onSuccess: () => void;
}

const isValidMethod = (method: string): boolean => {
  return ["GET", "POST", "HEAD", "PATCH", "DELETE"].indexOf(method) > -1;
};

const defaultParsed = { host: "", pathname: "" };

export default function TriggerFunctionModal({
  closeModal,
  onSuccess,
  triggerFunction,
  app,
  environment,
}: Props) {
  const options = triggerFunction?.options;
  const defHeaders = options?.headers || {};
  const parsed = options?.url ? new URL(options.url) : defaultParsed;
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState(Boolean(triggerFunction?.status));
  const { mode } = useContext(RootContext);
  const [host, setSelectedHost] = useState<string | undefined>(parsed.host);
  const [headers, setHeaders] = useState<Record<string, string>>(defHeaders);
  const [showHeaders, setShowHeaders] = useState(Boolean(options?.headers));
  const [showPayload, setShowPayload] = useState(Boolean(options?.payload));
  const [codeContent, setCodeContent] = useState(options?.payload || "");

  const defaultHeaders = useMemo(() => {
    return triggerFunction?.options?.headers || {};
  }, [triggerFunction?.options?.headers]);

  const handleSubmit: FormEventHandler = e => {
    e.preventDefault();
    e.stopPropagation();

    if (!host) {
      setError("Domain is a required field");
      return;
    }

    const form = e.target as HTMLFormElement;
    const values = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    const { urlPath = "", cron, method } = values;

    setLoading(true);
    setError(null);

    upsertFunctionTrigger({
      tfid: triggerFunction?.id,
      appId: app.id,
      envId: environment.id || "",
      cron,
      status,
      options: {
        url: `https://${host}/${urlPath.replace(/^\/+/, "")}`,
        method: method as FunctionTriggerMethod,
        headers,
        payload: codeContent,
      },
    })
      .then(() => {
        closeModal();
        onSuccess();
      })
      .catch((e: string | Error) => {
        if (typeof e === "string") {
          setError(e);
        } else if (typeof e === "object") {
          setError("Cron is invalid. Example expected format: 5 4 * * *");
        } else {
          setError("Something went wrong while saving periodic trigger");
        }
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Modal open onClose={closeModal}>
      <Card component="form" onSubmit={handleSubmit} error={error}>
        <CardHeader
          title={triggerFunction?.id ? "Edit trigger" : "Create trigger"}
        />
        <Box sx={{ mb: 4 }}>
          <DomainSelector
            variant="filled"
            label="Domain"
            placeholder="Select a domain"
            appId={app.id}
            envId={environment.id!}
            selected={host ? [host] : []}
            onDomainSelect={value => {
              if (Array.isArray(value) && typeof value[0] !== "string") {
                setSelectedHost(value[0]?.domainName);
              }
            }}
            multiple={false}
            withDevDomains={false}
            fullWidth
          />
        </Box>

        {host && (
          <>
            <Box sx={{ mb: 4 }}>
              <TextField
                name="urlPath"
                variant="filled"
                label="Path to trigger periodically"
                placeholder="/my-api-route"
                defaultValue={parsed.pathname || ""}
                inputProps={{
                  "aria-label": "URL to trigger periodically",
                }}
                InputLabelProps={{ shrink: true }}
                fullWidth
              />
            </Box>

            <Box sx={{ mb: 4 }}>
              <TextField
                name="cron"
                variant="filled"
                label="Cron"
                fullWidth
                placeholder="* * * * *"
                defaultValue={triggerFunction?.cron || ""}
                InputLabelProps={{ shrink: true }}
                helperText={
                  <>
                    Check{" "}
                    <Link
                      href="https://crontab.guru/"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      crontab.guru
                    </Link>{" "}
                    for expressions
                  </>
                }
              />
            </Box>
            <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
              <FormControlLabel
                sx={{ pl: 0, ml: 0 }}
                label="Enable headers"
                control={
                  <Switch
                    name="showHeaders"
                    color="secondary"
                    checked={showHeaders}
                    onChange={e => {
                      setShowHeaders(e.target.checked);
                    }}
                  />
                }
                labelPlacement="start"
              />
              <Typography sx={{ color: "text.secondary" }}>
                Turn on to send request headers
              </Typography>
            </Box>

            {showHeaders && (
              <Box data-testid="request-headers">
                <KeyValue
                  inputName="headers"
                  keyName="Header name"
                  valName="Header value"
                  keyPlaceholder="Content-Type"
                  valPlaceholder="application/json"
                  separator=":"
                  onChange={newVars => {
                    setHeaders(newVars);
                  }}
                  defaultValue={defaultHeaders}
                />
              </Box>
            )}

            <FormControl variant="standard" fullWidth sx={{ mb: 4 }}>
              <InputLabel id="request-method-label" sx={{ pl: 2, pt: 1 }}>
                Request method
              </InputLabel>
              <Select
                labelId="request-method-label"
                name="method"
                variant="filled"
                fullWidth
                defaultValue={triggerFunction?.options.method || "GET"}
                onChange={e => {
                  const value = e.target.value as FunctionTriggerMethod;

                  if (isValidMethod(value)) {
                    setShowPayload(value !== "GET");
                  }
                }}
              >
                <Option value={"GET"}>Get</Option>
                <Option value={"POST"}>Post</Option>
                <Option value={"HEAD"}>Head</Option>
                <Option value={"PATCH"}>Patch</Option>
                <Option value={"DELETE"}>Delete</Option>
              </Select>
            </FormControl>

            {showPayload && (
              <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
                <Typography variant="h4" sx={{ mb: 1 }}>
                  Request payload
                </Typography>
                <Form.Code
                  height="200px"
                  className="mb-4"
                  value={codeContent}
                  extensions={[json()]}
                  theme={mode}
                  onChange={value => setCodeContent(value)}
                />
              </Box>
            )}

            <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 2 }}>
              <FormControlLabel
                sx={{ pl: 0, ml: 0 }}
                label="Enable trigger"
                control={
                  <Switch
                    name="enabled"
                    color="secondary"
                    defaultChecked={triggerFunction?.status || false}
                    onChange={e => setStatus(e.target.checked)}
                  />
                }
                labelPlacement="start"
              />
              <Typography sx={{ color: "text.secondary" }}>
                Turn on to activate the trigger
              </Typography>
            </Box>
          </>
        )}

        <CardFooter>
          <Button type="button" variant="text" onClick={closeModal}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            sx={{ ml: 2 }}
            loading={loading}
          >
            {triggerFunction?.id ? "Update" : "Create"}
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
