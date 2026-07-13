import type { TriggerWhen, OutboundWebhook, AllowedMethod } from "../types";
import { useState, useEffect } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { grey } from "@mui/material/colors";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Select from "@mui/material/Select";
import Option from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import InputLabel from "@mui/material/InputLabel";
import Switch from "@mui/material/Switch";
import Typography from "@mui/material/Typography";
import KeyValue from "~/components/FormV2/KeyValue";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { upsertOutboundWebhook } from "../_actions/outbound_webhook_actions";
import Help from "~/components/Help/Help";

interface Props {
  app: App;
  isOpen: boolean;
  toggleModal: (val: boolean) => void;
  onUpdate: () => void;
  webhook?: OutboundWebhook;
}

const isValidMethod = (method: string): boolean => {
  return ["GET", "POST", "HEAD"].indexOf(method) > -1;
};

export default function FormNewOutboundWebhookModal({
  isOpen,
  toggleModal,
  onUpdate,
  app,
  webhook,
}: Props) {
  const [trigger, setTrigger] = useState<TriggerWhen>("on_deploy_success");
  const [method, setMethod] = useState("GET");
  const [payload, setPayload] = useState("");
  const [headers, setHeaders] = useState<Record<string, string>>({});
  const [showHeaders, setShowHeaders] = useState(false);
  const [showPayload, setShowPayload] = useState(false);
  const [error, setError] = useState<null | string>(null);
  const [loading, setLoading] = useState<boolean>(false);

  useEffect(() => {
    let unmounted = false;

    if (isOpen === false) {
      setTimeout(() => {
        if (unmounted === false) {
          setError(null);
          setShowPayload(false);
          setShowHeaders(false);
          setHeaders({});
          setMethod("GET");
          setTrigger("on_deploy_success");
        }
      }, 250);
    }

    return () => {
      unmounted = true;
    };
  }, [isOpen]);

  useEffect(() => {
    if (webhook) {
      setMethod(webhook.requestMethod);
      setTrigger(webhook.triggerWhen);
      setPayload(webhook.requestPayload || "");
      setHeaders(webhook?.requestHeaders || {});
    }
  }, [webhook]);

  useEffect(() => {
    setShowPayload(method === "POST");
  }, [method]);

  useEffect(() => {
    setShowHeaders(Object.keys(headers).length > 0);
  }, [headers]);

  return (
    <Modal
      open={isOpen}
      onClose={() => {
        toggleModal(false);
      }}
    >
      <Card
        component="form"
        error={error}
        onSubmit={e => {
          e.preventDefault();

          const form = e.target as HTMLFormElement;
          const data = Object.fromEntries(
            new FormData(form).entries()
          ) as Record<string, string>;

          if (!isValidMethod(data.requestMethod)) {
            setError(
              "Invalid method provided. Can be one of: GET | POST | HEAD"
            );

            return;
          }

          setLoading(true);
          setError(null);

          upsertOutboundWebhook({
            app,
            id: data.id,
            requestUrl: data.requestUrl,
            requestMethod: data.requestMethod as AllowedMethod,
            requestPayload: data.requestMethod !== "POST" ? undefined : payload,
            requestHeaders: headers,
            triggerWhen: data.triggerWhen as TriggerWhen,
          })
            .then(() => {
              setError(null);
              toggleModal(false);
              onUpdate();
            })
            .catch(e => {
              if (e.status === 400) {
                setError("Please make sure to provide a valid URL.");
              } else {
                setError(
                  "Something went wrong while creating new webhook. Please contact us or try again later."
                );
              }
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title={`${webhook ? "Update" : "Create an"} outbound webhook`}
          subtitle={
            <>
              Check out the{" "}
              <Link href="https://www.stormkit.io/docs/deployments/outbound-webhooks">
                documentation
              </Link>{" "}
              for examples
            </>
          }
        />
        <TextField
          type="hidden"
          defaultValue={webhook?.id}
          name="id"
          sx={{ display: "none" }}
        />
        <Box sx={{ mb: 4 }}>
          <TextField
            label="Request URL"
            variant="filled"
            autoComplete="off"
            defaultValue={webhook?.requestUrl || ""}
            fullWidth
            name="requestUrl"
            placeholder="https://example.org/webhooks"
            helperText="The fully qualified URL where the request will be made."
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
            Turn on to send request headers.
          </Typography>
        </Box>
        {showHeaders && (
          <Box data-testid="request-headers" sx={{ mb: 4 }}>
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
              defaultValue={headers}
            />
          </Box>
        )}

        <FormControl variant="standard" fullWidth sx={{ mb: 4 }}>
          <InputLabel id="request-method-label" sx={{ pl: 2, pt: 1 }}>
            Request method
          </InputLabel>
          <Select
            labelId="request-method-label"
            name="requestMethod"
            variant="filled"
            value={method}
            fullWidth
            defaultValue="GET"
            onChange={e => {
              const value = e.target.value as "GET" | "POST" | "HEAD";

              if (isValidMethod(value)) {
                setShowPayload(value === "POST");
                setMethod(value);
              }
            }}
          >
            <Option value={"GET"}>Get</Option>
            <Option value={"POST"}>Post</Option>
            <Option value={"HEAD"}>Head</Option>
          </Select>
        </FormControl>

        {showPayload && (
          <Box
            sx={{
              mb: 4,
              p: 2,
              bgcolor: "container.paper",
              borderBottom: `1px solid ${grey[900]}`,
            }}
          >
            <Typography sx={{ mb: 2, color: "text.secondary" }}>
              Request payload
            </Typography>
            <CodeMirror
              maxHeight="200px"
              value={payload}
              extensions={[json()]}
              onChange={v => setPayload(v)}
              theme="dark"
            />
          </Box>
        )}

        <FormControl variant="standard" fullWidth sx={{ mb: 4 }}>
          <InputLabel id="trigger-when-label" sx={{ pl: 2, pt: 1 }}>
            Trigger when{" "}
          </InputLabel>
          <Select
            labelId="trigger-when-label"
            name="triggerWhen"
            variant="filled"
            value={trigger}
            fullWidth
            onChange={e => {
              if (typeof e.target.value === "string") {
                setTrigger(e.target.value as TriggerWhen);
              }
            }}
          >
            <Option value={"on_deploy_success"}>
              After each successful deployment
            </Option>
            <Option value={"on_deploy_failed"}>
              After each failed deployment
            </Option>
            <Option value={"on_publish"}>After deployment is published</Option>
            <Option value={"on_cache_purge"}>After cache is purged</Option>
          </Select>
        </FormControl>

        <CardFooter sx={{ display: "flex", justifyContent: "space-between" }}>
          <Help
            title="Outbound webhooks"
            subtitle="Automate workflows by sending HTTP requests to external services when specific events occur in your application deployment lifecycle."
          >
            <Typography variant="subtitle1" sx={{ fontWeight: "bold", mb: 1 }}>
              Overview
            </Typography>
            <Typography color="text.secondary" sx={{ mb: 2 }}>
              Outbound webhooks are HTTP requests that are triggered by specific
              events in your application deployment lifecycle. You can configure
              the request method, headers, and payload to suit your needs.
            </Typography>
            <Typography variant="subtitle1" sx={{ fontWeight: "bold", mb: 1 }}>
              Events
            </Typography>

            <Typography variant="subtitle1" sx={{ fontWeight: "bold", mb: 1 }}>
              1. After each successful deployment
            </Typography>
            <Typography color="text.secondary">
              The webhook will be triggered after every successful deployment of
              your application. A successful deployment means that your code has
              been built and deployed without any errors. This event cycle
              occurs before status checks are run.
            </Typography>
            <Typography
              variant="subtitle1"
              sx={{ fontWeight: "bold", mt: 2, mb: 1 }}
            >
              2. After each failed deployment
            </Typography>
            <Typography color="text.secondary">
              The webhook will be triggered after every failed deployment. This
              event cycle occurs before status checks are run.
            </Typography>
            <Typography
              variant="subtitle1"
              sx={{ fontWeight: "bold", mt: 2, mb: 1 }}
            >
              3.After a deployment is published
            </Typography>
            <Typography color="text.secondary">
              The webhook will be triggered after a deployment is published.
              This event cycle occurs after status checks are run.
            </Typography>
            <Typography
              variant="subtitle1"
              sx={{ fontWeight: "bold", mt: 2, mb: 1 }}
            >
              4.After a cache purge
            </Typography>
            <Typography color="text.secondary">
              Cache is purged after the following events:
              <br />
              <br />- A deployment is published
              <br />- Any operation to snippets
              <br />- The environment configuration is updated
            </Typography>
          </Help>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={loading}
          >
            {webhook ? "Update" : "Create"} outbound webhook
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
