import type { SendSampleRequestResponse } from "../_actions/outbound_webhook_actions";
import type { OutboundWebhook, TriggerWhen } from "../types";
import React, { useMemo, useState } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import Modal from "~/components/Modal";
import Spinner from "~/components/Spinner";
import DotDotDot from "~/components/DotDotDotV2";
import { truncate } from "~/utils/helpers/string";
import FormNewOutboundWebhookModal from "./FormOutboundWebhookModal";
import {
  useFetchOutboundWebhooks,
  sendSampleRequest,
  deleteOutboundWebhook,
} from "../_actions/outbound_webhook_actions";
import { grey } from "@mui/material/colors";

interface Props {
  app: App;
}

const messages: Record<TriggerWhen, string> = {
  on_deploy_success: "Triggered after each successful deployment",
  on_deploy_failed: "Triggered after each failed deployment",
  on_publish: "Triggered after a deployment is published",
  on_cache_purge: "Triggered after a cache purge",
};

const FormOutboundWebhooks: React.FC<Props> = ({ app }): React.ReactElement => {
  const [loadingSample, setLoadingSample] = useState(false);
  const [webhookToEdit, setWebhookToEdit] = useState<OutboundWebhook>();
  const [sampleData, setSampleData] = useState<SendSampleRequestResponse>();
  const [isModalOpen, toggleModal] = useState(false);
  const [refreshToken, setRefreshToken] = useState<number>();
  const { hooks, error, loading } = useFetchOutboundWebhooks({
    app,
    refreshToken,
  });

  const sampleResponse = useMemo(() => {
    try {
      return JSON.stringify(JSON.parse(sampleData?.result.body || ""), null, 4);
    } catch {
      return "Empty response";
    }
  }, [sampleData?.result.body]);

  return (
    <Card error={error} sx={{ mb: 4 }}>
      <CardHeader
        title="Outbound webhooks"
        subtitle="Integrate external applications with Stormkit using webhooks."
      />
      <Box sx={{ mb: 4 }}>
        {hooks.length ? (
          hooks.map(hook => (
            <Box
              key={hook.id}
              sx={{
                bgcolor: "rgba(255,255,255,0.015)",
                border: `1px solid ${grey[900]}`,
                display: "flex",
                alignItems: "center",
                mb: 2,
                p: 2,
              }}
            >
              <Box sx={{ flex: 1 }}>
                <Typography sx={{ fontWeight: "bold" }}>
                  {hook.requestMethod?.toUpperCase()}
                </Typography>
                <Typography>{truncate(hook.requestUrl, 50)}</Typography>
                <Typography>{messages[hook.triggerWhen]}</Typography>
              </Box>
              <div className="self-baseline">
                <DotDotDot
                  items={[
                    {
                      text: "Edit",
                      onClick: () => {
                        setWebhookToEdit(hook);
                        toggleModal(true);
                      },
                    },
                    {
                      text: (
                        <Box sx={{ display: "flex", alignItems: "center" }}>
                          Trigger sample
                          {loadingSample && (
                            <Spinner
                              className="ml-2"
                              primary
                              width={4}
                              height={4}
                            />
                          )}
                        </Box>
                      ),
                      onClick: close => {
                        setLoadingSample(true);

                        sendSampleRequest({ app, whId: hook.id })
                          .then(res => {
                            setLoadingSample(false);
                            setSampleData(res);
                          })
                          .finally(() => {
                            close?.();
                          });

                        return false;
                      },
                    },
                    {
                      text: "Delete",
                      onClick: close => {
                        deleteOutboundWebhook({
                          app,
                          whId: hook.id,
                        }).then(() => {
                          setRefreshToken(Date.now());
                          close?.();
                        });
                      },
                    },
                  ]}
                />
              </div>
            </Box>
          ))
        ) : (
          <Alert color="info">
            <Typography>You don't have any webhooks installed.</Typography>
          </Alert>
        )}
      </Box>
      <CardFooter>
        <Button
          color="secondary"
          variant="contained"
          type="button"
          loading={loading}
          onClick={() => {
            toggleModal(true);
          }}
        >
          Add new webhook
        </Button>
      </CardFooter>
      <Modal
        open={Boolean(sampleData)}
        onClose={() => {
          setSampleData(undefined);
        }}
      >
        <Card>
          <CardHeader title={`Sample result: ${sampleData?.result.status}`} />
          <Box sx={{ mb: 4 }}>
            <Box
              component="pre"
              sx={{
                color: "text.primary",
                bgcolor: "container.transparent",
                fontFamily: "monospace",
                fontSize: 12,
                p: 2,
              }}
            >
              {sampleResponse}
            </Box>
          </Box>
          <CardFooter>
            <Button
              type="button"
              onClick={() => {
                setSampleData(undefined);
              }}
              variant="contained"
              color="secondary"
            >
              Close
            </Button>
          </CardFooter>
        </Card>
      </Modal>

      {isModalOpen && (
        <FormNewOutboundWebhookModal
          app={app}
          isOpen
          onUpdate={() => {
            setRefreshToken(Date.now());
          }}
          toggleModal={(val: boolean) => {
            toggleModal(val);
            setWebhookToEdit(undefined);
          }}
          webhook={webhookToEdit}
        />
      )}
    </Card>
  );
};

export default FormOutboundWebhooks;
