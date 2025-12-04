import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import CircularProgress from "@mui/material/CircularProgress";
import CheckIcon from "@mui/icons-material/Check";
import TimesIcon from "@mui/icons-material/Close";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import Api from "~/utils/api/Api";
import KeyValue from "~/components/FormV2/KeyValue";
import Card from "~/components/Card";
import CardRow from "~/components/CardRow";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";

type Status = "ok" | "sent" | "processing" | "error";

interface PackageOutput {
  version: string;
  requested_version: string;
  active: boolean;
  installed: boolean;
}

type Installed = Record<string, PackageOutput[]>;

interface FetchRuntimesResponse {
  runtimes: string[];
  installed: Installed;
  autoInstall: boolean;
  status: Status;
}

const useFetchRuntimes = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [runtimes, setRuntimes] = useState<string[]>();
  const [installed, setInstalled] = useState<Installed>();
  const [autoInstall, setAutoInstall] = useState(true);
  const [status, setStatus] = useState<Status>();
  const [refreshToken, setRefreshToken] = useState<number>();

  let timeout: NodeJS.Timeout;

  useEffect(() => {
    Api.fetch<FetchRuntimesResponse>("/admin/system/runtimes")
      .then(({ runtimes, autoInstall, installed, status: s }) => {
        setRuntimes(runtimes);
        setAutoInstall(autoInstall);
        setStatus(s);
        setInstalled(installed);

        console.log("Runtimes status:", runtimes);

        if (s !== "ok" && s !== "error") {
          clearTimeout(timeout);
          timeout = setTimeout(() => {
            setRefreshToken(Date.now());
          }, 2500);
        }
      })
      .catch(() => setError("Something went wrong while fetching runtimes"))
      .finally(() => setLoading(false));
  }, [refreshToken]);

  return {
    loading,
    error,
    runtimes,
    status,
    installed,
    autoInstall,
    setRefreshToken,
  };
};

export const mapRuntimes = (runtimes: string[]): Record<string, string> => {
  const result: Record<string, string> = {};

  runtimes.forEach(runtime => {
    // Find the last @ symbol to split package name from version
    const lastAtIndex = runtime.lastIndexOf("@");

    if (lastAtIndex === -1) {
      // No @ found, treat entire string as package name with undefined version
      result[runtime] = "latest";
    } else {
      // Normal case: split at the last @
      const packageName = runtime.substring(0, lastAtIndex);
      const version = runtime.substring(lastAtIndex + 1);
      result[packageName] = version;
    }
  });

  return result;
};

export const mapRuntimeStatus = (
  status: Status,
  installed: Installed,
  runtimes: Record<string, string>
): React.ReactNode | React.ReactNode[] => {
  if (
    status === "sent" ||
    status === "processing" ||
    !status ||
    !installed ||
    !runtimes
  ) {
    return <CircularProgress size={14} />;
  }

  return Object.keys(runtimes).map(runtime => {
    const runtimeStatus = installed[runtime]?.find(
      pkg => pkg.requested_version === runtimes[runtime]
    );

    // Placeholder icon
    if (!runtimeStatus && status === "error" && runtimes[runtime]) {
      return <TimesIcon key={runtime} color="error" sx={{ fontSize: 14 }} />;
    }

    if (runtimeStatus?.active) {
      return <CheckIcon key={runtime} color="success" sx={{ fontSize: 14 }} />;
    }

    return (
      <CheckIcon
        key={runtime}
        color="success"
        sx={{ visibility: "hidden", fontSize: 14 }}
      />
    );
  });
};

function Runtimes() {
  const {
    error,
    loading,
    runtimes,
    status: installStatus,
    installed,
    autoInstall: auto,
    setRefreshToken,
  } = useFetchRuntimes();

  const [updateError, setUpdateError] = useState<string>();
  const [updateLoading, setUpdateLoading] = useState(false);
  const [kv, setKV] = useState<Record<string, string>>({});
  const [autoInstall, setAutoInstall] = useState(auto);

  useEffect(() => {
    setKV(mapRuntimes(runtimes || []));
  }, [runtimes]);

  useEffect(() => {
    setAutoInstall(auto);
  }, [auto]);

  return (
    <Card
      loading={loading}
      error={
        error ||
        updateError ||
        (installStatus === "error" &&
          "An error occurred while installing runtimes. Check instance logs for more details.")
      }
      sx={{ backgroundColor: "container.transparent" }}
      contentPadding={false}
    >
      <CardHeader
        title="Installed runtimes"
        subtitle="Manage runtimes that are installed on your Stormkit instance"
      />
      <Box sx={{ px: 4 }}>
        <KeyValue
          defaultValue={kv}
          inputName="runtimes"
          keyName="Runtime name"
          valName="Runtime version"
          keyIcons={mapRuntimeStatus(installStatus!, installed!, kv)}
          keyPlaceholder="node"
          valPlaceholder="24"
          modifyAsString={false}
          onChange={newVars => {
            setKV(newVars);
            setUpdateError(undefined);
          }}
        />
      </Box>
      <Box
        sx={{ bgcolor: "container.transparent", p: 1.75, pt: 1, mb: 4, mx: 4 }}
      >
        <FormControlLabel
          sx={{ pl: 0, ml: 0 }}
          label="Auto install"
          control={
            <Switch
              name="autoInstall"
              color="secondary"
              checked={autoInstall}
              onChange={e => {
                setAutoInstall(e.target.checked);
              }}
            />
          }
          labelPlacement="start"
        />
        <Typography color="text.secondary">
          When enabled, runtimes are automatically installed during deployment
          based on your app's version configuration files.
          <br />
          Check the{" "}
          <Link
            href="https://www.stormkit.io/docs/self-hosting/runtimes"
            color="error"
            target="_blank"
            rel="noopener noreferrer"
          >
            documentation
          </Link>{" "}
          for details.
        </Typography>
      </Box>
      <CardFooter>
        <Button
          variant="contained"
          color="secondary"
          type="submit"
          loading={
            updateLoading ||
            installStatus === "processing" ||
            installStatus === "sent"
          }
          onClick={() => {
            setUpdateLoading(true);

            Api.post("/admin/system/runtimes", {
              autoInstall,
              runtimes: Object.entries(kv).map(
                ([key, value]) => `${key}@${value || "latest"}`
              ),
            })
              .then(() => {
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setUpdateError(
                  "An error occurred while installing runtimes. Make sure specified versions are correct."
                );
              })
              .finally(() => {
                setUpdateLoading(false);
              });
          }}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

const useFetchMise = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [refreshToken, setRefreshToken] = useState(0);
  const [version, setVersion] = useState<string>();
  const [status, setStatus] = useState<Status>();

  let timeout: NodeJS.Timeout;

  useEffect(() => {
    Api.fetch<{ version: string; status: Status }>("/admin/system/mise")
      .then(({ version, status: s }) => {
        setVersion(version);
        setLoading(s === "sent" || s === "processing");
        setStatus(s);

        if (s !== "ok" && s !== "error") {
          clearTimeout(timeout);
          timeout = setTimeout(() => {
            setRefreshToken(Date.now());
          }, 2500);
        }

        if (s === "error") {
          setError(
            "An error occurred while installing or upgrading mise. Check instance logs for more details."
          );
        }
      })
      .catch(() => {
        setError("Something went wrong while fetching mise version");
        setLoading(false);
      });
  }, [refreshToken]);

  return { loading, error, version, status, setRefreshToken };
};

function Mise() {
  const { error, loading, version, status, setRefreshToken } = useFetchMise();
  const [updateError, setUpdateError] = useState<string>();
  const [updateLoading, setUpdateLoading] = useState(false);

  return (
    <Card
      error={error || updateError}
      sx={{ mt: 4, backgroundColor: "container.transparent" }}
      contentPadding={false}
    >
      <CardHeader
        title="Mise"
        subtitle="Stormkit relies on open-source mise for runtime management"
        actions={
          <Button
            variant="contained"
            color="secondary"
            loading={
              updateLoading || status === "processing" || status === "sent"
            }
            onClick={() => {
              setUpdateLoading(true);

              Api.post<{ status: "error" | "ok" }>("/admin/system/mise")
                .then(({ status }) => {
                  if (status === "error") {
                    setUpdateError(
                      "An error occurred while upgrading mise. Check instance logs for more details."
                    );

                    return;
                  }

                  setRefreshToken(Date.now());
                })
                .catch(() => {
                  setUpdateError("An error occurred while upgrading mise.");
                })
                .finally(() => {
                  setUpdateLoading(false);
                });
            }}
          >
            Upgrade to latest
          </Button>
        }
      />
      <CardRow>
        <Typography variant="h2" color="text.secondary" sx={{ mb: 1 }}>
          Current version
        </Typography>
        <Typography sx={{ display: "flex", alignItems: "center" }}>
          {status === "ok" ? (
            <CheckIcon color="success" sx={{ fontSize: 14, mr: 1 }} />
          ) : (
            loading && <CircularProgress size={14} sx={{ mr: 1 }} />
          )}
          {version || "Unknown"}
        </Typography>
      </CardRow>
    </Card>
  );
}

interface DomainsConfig {
  dev: string;
  api: string;
  app: string;
}

const useFetchDomains = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [domains, setDomains] = useState<DomainsConfig>({
    dev: "",
    api: "",
    app: "",
  });

  useEffect(() => {
    Api.fetch<{ domains: DomainsConfig }>("/admin/domains")
      .then(({ domains }) => {
        setDomains(domains);
      })
      .catch(() => {
        setError("Something went wrong while fetching domains");
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return { loading, error, domains };
};

function Domains() {
  const { error, loading, domains } = useFetchDomains();
  const [updateError, setUpdateError] = useState<string>();
  const [updateLoading, setUpdateLoading] = useState(false);

  return (
    <Card
      error={error || updateError}
      loading={loading}
      sx={{ mt: 4, backgroundColor: "container.transparent" }}
      contentPadding={false}
      component="form"
    >
      <CardHeader
        title="Domains"
        subtitle="Configure custom domains for your Stormkit instance"
      />
      <CardRow>
        <TextField
          label="API Domain"
          variant="filled"
          name="api"
          defaultValue={domains.api}
          slotProps={{ inputLabel: { shrink: true } }}
          helperText="API requests will be served from this domain"
          fullWidth
        />
      </CardRow>
      <CardRow>
        <TextField
          label="App Domain"
          name="app"
          defaultValue={domains.app}
          variant="filled"
          slotProps={{ inputLabel: { shrink: true } }}
          helperText="This domain will be used to access your Stormkit dashboard"
          fullWidth
        />
      </CardRow>
      <CardRow>
        <TextField
          label="Dev Domain"
          name="dev"
          defaultValue={domains.dev}
          variant="filled"
          slotProps={{ inputLabel: { shrink: true } }}
          helperText="Deployment previews will be displayed using subdomains of this domain"
          fullWidth
        />
      </CardRow>
      <CardFooter>
        <Button
          variant="contained"
          color="secondary"
          type="submit"
          loading={updateLoading}
          onClick={e => {
            e.preventDefault();
            setUpdateLoading(true);

            const form = e.currentTarget.form as HTMLFormElement;
            const formData = new FormData(form);
            const payload = {
              api: formData.get("api") as string,
              app: formData.get("app") as string,
              dev: formData.get("dev") as string,
            };

            Api.post<DomainsConfig>("/admin/domains", payload)
              .then(() => {
                window.location.reload();
              })
              .catch(() => {
                setUpdateError(
                  "An error occurred while updating domains. Make sure specified domains are valid."
                );
              })
              .finally(() => {
                setUpdateLoading(false);
              });
          }}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

export default function System() {
  return (
    <Box>
      <Runtimes />
      <Mise />
      <Domains />
    </Box>
  );
}
