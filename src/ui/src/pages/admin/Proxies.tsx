import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import ArrowForward from "@mui/icons-material/ArrowForward";
import Modal from "~/components/Modal";
import KeyValue from "~/components/FormV2/KeyValue";
import Card from "~/components/Card";
import CardRow from "~/components/CardRow";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import api from "~/utils/api/Api";

interface ProxyRule {
  target: string;
  headers?: Record<string, string>;
  domain?: string;
}

interface ProxyConfig {
  rules: Record<string, ProxyRule>;
}

interface UseFetchProxiesProps {
  refreshToken?: number;
}

const useFetchProxies = ({ refreshToken }: UseFetchProxiesProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [proxies, setProxies] = useState<ProxyConfig>();

  useEffect(() => {
    api
      .fetch<{ proxies: ProxyConfig }>("/admin/system/proxies")
      .then(({ proxies }) => {
        setProxies(proxies);
      })
      .catch(() => setError("Something went wrong while fetching proxies"))
      .finally(() => setLoading(false));
  }, [refreshToken]);

  return { loading, error, proxies };
};

interface ProxyModalProps {
  onClose: () => void;
  onSuccess: () => void;
  proxy?: ProxyRule;
}

function ProxyModal({ onClose, onSuccess, proxy }: ProxyModalProps) {
  const [headers, setHeaders] = useState<Record<string, string>>(
    proxy?.headers || {}
  );

  const [updateLoading, setUpdateLoading] = useState(false);
  const [updateError, setUpdateError] = useState<string>();

  return (
    <Modal open maxWidth="800px" onClose={onClose}>
      <Card
        error={updateError}
        component="form"
        onSubmit={e => {
          e.preventDefault();

          const form = e.target as HTMLFormElement;
          const data = Object.fromEntries(
            new FormData(form).entries()
          ) as Record<string, string>;

          const from = data.from.replace(/^(https?:)?\/\//, "").trim();

          if (!from) {
            setUpdateError("From field is required");
            return;
          }

          if (!data.target.trim()) {
            setUpdateError("Target field is required");
            return;
          }

          if (!data.target.match(/^(https?:)?\/\//)) {
            setUpdateError(
              "Target must be a fully qualified URL (e.g. http://example.com)"
            );
            return;
          }

          setUpdateLoading(true);

          api
            .put("/admin/system/proxies", {
              proxies: {
                [from]: {
                  target: data.target,
                  headers,
                },
              },
            })
            .then(() => {
              onSuccess();
            })
            .catch(() => {
              setUpdateError("Something went wrong while updating the proxy");
            })
            .finally(() => {
              setUpdateLoading(false);
            });
        }}
      >
        <CardHeader
          title={proxy ? "Update proxy" : "New proxy"}
          subtitle={
            proxy
              ? "Update existing proxy to forward requests to a 3rd party application"
              : "Create a new reverse proxy to forward requests to a 3rd party application"
          }
        />
        <Box>
          <TextField
            label="From"
            variant="filled"
            name="from"
            fullWidth
            autoFocus
            defaultValue={proxy?.domain || ""}
            placeholder="example.com"
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
            helperText={
              <Typography variant="caption">
                The domain to forward requests from. Stormkit will issue TLS
                certificates automatically for this domain.
              </Typography>
            }
            sx={{ mb: 4 }}
          />

          <TextField
            label="Target"
            variant="filled"
            name="target"
            defaultValue={proxy?.target || ""}
            fullWidth
            placeholder="http://localhost:3000"
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
            helperText={
              <Typography variant="caption">
                The URL to forward requests to. If behind a firewall, make sure
                the target is accessible from Stormkit servers.
              </Typography>
            }
            sx={{ mb: 4 }}
          />

          <KeyValue
            defaultValue={headers}
            inputName="headers"
            keyName="Header name"
            valName="Header value"
            keyPlaceholder="X-Forwarded-For"
            valPlaceholder="203.0.113.195"
            separator=":"
            modifyAsString={true}
            onChange={newVars => {
              setHeaders(newVars);
            }}
          />
        </Box>
        <CardFooter>
          <Button color="primary" onClick={onClose} sx={{ mr: 2 }}>
            Close
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            loading={updateLoading}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}

export default function Proxies() {
  const [refreshToken, setRefreshToken] = useState(0);
  const { error, loading, proxies } = useFetchProxies({ refreshToken });
  const [proxyToBeUpdated, setProxyToBeUpdated] = useState<ProxyRule>();
  const [proxyToBeDeleted, setProxyToBeDeleted] = useState<string>();
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <Card
      loading={loading}
      error={error}
      sx={{ backgroundColor: "container.transparent" }}
      contentPadding={false}
      info={
        !loading && Object.keys(proxies?.rules || {}).length === 0
          ? "No proxy configured yet."
          : undefined
      }
    >
      <CardHeader
        title="Reverse proxies"
        subtitle={
          <>
            Configure reverse proxies for your 3rd party applications.
            <br />
            Stormkit will issue TLS certificates automatically for these proxied
            domains.
          </>
        }
        actions={
          <Button
            variant="contained"
            color="secondary"
            onClick={() => {
              setProxyToBeUpdated(undefined);
              setIsModalOpen(true);
            }}
          >
            New proxy
          </Button>
        }
      />
      {proxies &&
        Object.keys(proxies.rules).map(domain => (
          <CardRow
            menuItems={[
              {
                text: "Edit",
                onClick: () => {
                  setProxyToBeUpdated({ ...proxies.rules[domain], domain });
                  setIsModalOpen(true);
                },
              },
              {
                text: "Delete",
                onClick: () => {
                  setProxyToBeDeleted(domain);
                },
              },
            ]}
            key={domain}
          >
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
              {domain}
              <ArrowForward sx={{ mx: 1, fontSize: 12 }} />
              {proxies.rules[domain].target}
            </Typography>
          </CardRow>
        ))}

      {isModalOpen && (
        <ProxyModal
          proxy={proxyToBeUpdated}
          onSuccess={() => {
            setIsModalOpen(false);
            setRefreshToken(Date.now());
          }}
          onClose={() => {
            setIsModalOpen(false);
          }}
        />
      )}
      {proxyToBeDeleted && (
        <ConfirmModal
          onCancel={() => {
            setProxyToBeDeleted(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            api
              .put("/admin/system/proxies", {
                remove: [proxyToBeDeleted],
              })
              .then(() => {
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setError(
                  "Something went wrong while deleting the proxy configuration."
                );
              })
              .finally(() => {
                setLoading(false);
                setProxyToBeDeleted(undefined);
              });
          }}
        >
          This will delete the proxy rule. Previously configured domain may stop
          working.
        </ConfirmModal>
      )}
      <CardFooter />
    </Card>
  );
}
