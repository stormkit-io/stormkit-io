import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import FormControlLabel from "@mui/material/FormControlLabel";
import Link from "@mui/material/Link";
import Switch from "@mui/material/Switch";
import Button from "@mui/material/Button";
import LaunchIcon from "@mui/icons-material/Launch";
import Typography from "@mui/material/Typography";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";
import { GitDetails } from "./types.d";

interface Props {
  details?: GitDetails;
  closeModal: () => void;
  onSuccess: (message: string) => void;
}

export default function GitHubModal({ closeModal, onSuccess, details }: Props) {
  const gh = details?.github;
  const [isManual, setIsManual] = useState(Boolean(gh));
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [appName, setAppName] = useState(gh?.account || "");
  const [appId, setAppId] = useState(gh?.appId || "");
  const [clientId, setClientId] = useState(gh?.clientId || "");
  const [clientSecret, setClientSecret] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [isOrganization, setIsOrganization] = useState(false);
  const [organizationName, setOrganizationName] = useState("");
  const [redirectURL, setRedirectURL] = useState("");
  const [manifest, setManifest] = useState("");

  useEffect(() => {
    if (!manifest) {
      return;
    }

    const form = document.getElementById("github-app-form") as HTMLFormElement;

    if (form) {
      form.submit();
    }
  }, [manifest]);

  return (
    <Modal open onClose={closeModal} maxWidth="md">
      <Card
        component="form"
        error={error}
        onSubmit={e => {
          e.preventDefault();

          if (!appName.trim()) {
            setError("App name is required");
            return;
          }

          if (!isManual && isOrganization && !organizationName.trim()) {
            setError(
              "Organization name is required when organization is enabled"
            );
            return;
          }

          setLoading(true);
          setError(undefined);

          const payload = isManual
            ? {
                provider: "github",
                appId: appId.trim(),
                account: appName.trim(),
                clientId: clientId.trim(),
                clientSecret: clientSecret.trim() || undefined,
                privateKey: privateKey,
              }
            : {
                appName: appName.trim(),
                organization: isOrganization
                  ? organizationName.trim()
                  : undefined,
              };

          let promise: Promise<any>;

          if (isManual) {
            promise = api.post("/admin/git/configure", payload);
          } else {
            promise = api.post<{ manifest: Record<string, any>; url: string }>(
              "/admin/git/github/manifest",
              payload
            );
          }

          promise
            .then(({ manifest, url }) => {
              if (manifest) {
                setManifest(JSON.stringify(manifest));
                setRedirectURL(url);
              } else {
                onSuccess(
                  "GitHub configuration updated. Your browser will be refreshed for changes to take effect."
                );
                closeModal();
              }
            })
            .catch(res => {
              if (res?.error) {
                setError(res.error);
              } else {
                setError(
                  isManual
                    ? "Something went wrong while configuring GitHub App"
                    : "Something went wrong while creating GitHub App manifest"
                );
              }
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title="Configure GitHub"
          subtitle={
            <>
              Configure your Github app to access private repositories and
              authenticate users via GitHub oAuth.
              <br />
              {isManual ? (
                <>
                  This flow is for manual configuration.{" "}
                  <Link color="secondary" onClick={() => setIsManual(false)}>
                    Click here
                  </Link>{" "}
                  to create a new app instead.
                </>
              ) : (
                <>
                  This is an automated flow. If you wish to configure it
                  manually{" "}
                  <Link
                    color="secondary"
                    onClick={() => setIsManual(!isManual)}
                  >
                    click here
                  </Link>{" "}
                  instead.
                </>
              )}
            </>
          }
        />

        {manifest && redirectURL && (
          <Box
            component="form"
            action={redirectURL}
            method="post"
            id="github-app-form"
            sx={{ visibility: "hidden" }}
          >
            <input
              type="hidden"
              name="manifest"
              id="manifest"
              value={manifest}
            />
          </Box>
        )}

        <TextField
          label="App Name"
          variant="filled"
          placeholder="Name your GitHub App"
          value={appName}
          onChange={e => setAppName(e.target.value)}
          fullWidth
          required
          sx={{ mb: 4 }}
          slotProps={{
            inputLabel: {
              shrink: true,
            },
          }}
        />

        <Box
          sx={{
            bgcolor: "container.paper",
            p: 1.75,
            pt: 1,
            mb: 4,
            display: isManual ? "none" : undefined,
          }}
        >
          <FormControlLabel
            sx={{ pl: 0, ml: 0 }}
            label="Organization"
            control={
              <Switch
                checked={isOrganization}
                onChange={e => setIsOrganization(e.target.checked)}
                color="secondary"
              />
            }
            labelPlacement="start"
          />
          <Typography color="text.secondary">
            Enable if the GitHub App is going to be installed on an organization
          </Typography>
        </Box>

        {isOrganization && !isManual && (
          <TextField
            label="Organization Name"
            placeholder="Enter organization name"
            value={organizationName}
            onChange={e => setOrganizationName(e.target.value)}
            fullWidth
            required
            variant="filled"
            sx={{ mb: 4 }}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />
        )}

        {isManual && (
          <>
            <TextField
              label="App ID"
              variant="filled"
              placeholder="The App ID of your GitHub App (e.g. 123456)"
              value={appId}
              onChange={e => setAppId(e.target.value)}
              fullWidth
              required
              sx={{ mb: 4 }}
              slotProps={{
                inputLabel: {
                  shrink: true,
                },
              }}
            />

            <TextField
              label="Client ID"
              variant="filled"
              placeholder="The client ID of your GitHub App (e.g. Iv1.1234567890abcdef)"
              value={clientId}
              onChange={e => setClientId(e.target.value)}
              fullWidth
              required
              sx={{ mb: 4 }}
              slotProps={{
                inputLabel: {
                  shrink: true,
                },
              }}
            />

            <TextField
              label="Client secret"
              variant="filled"
              placeholder={
                gh?.hasClientSecret
                  ? "***************"
                  : "The client secret of your GitHub App"
              }
              value={clientSecret}
              onChange={e => setClientSecret(e.target.value)}
              fullWidth
              required
              sx={{ mb: 4 }}
              slotProps={{
                inputLabel: {
                  shrink: true,
                },
              }}
            />

            <TextField
              label="Private key"
              variant="filled"
              placeholder={
                gh?.hasPrivateKey
                  ? "***************"
                  : "The private key of your GitHub App"
              }
              value={privateKey}
              onChange={e => setPrivateKey(e.target.value)}
              fullWidth
              required
              multiline
              minRows={privateKey ? Math.ceil(privateKey.length / 100) : 1}
              maxRows={10}
              sx={{ mb: 4 }}
              slotProps={{
                inputLabel: {
                  shrink: true,
                },
              }}
            />
          </>
        )}

        <CardFooter
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Button type="button" sx={{ visibility: "hidden" }}>
            Delete
          </Button>
          <Box>
            <Button onClick={closeModal} disabled={loading}>
              Cancel
            </Button>
            <Button
              variant="contained"
              color="secondary"
              type="submit"
              disabled={loading}
              loading={loading}
              sx={{ ml: 2 }}
              endIcon={isManual ? undefined : <LaunchIcon />}
            >
              {isManual ? "Configure" : "Continue on Github"}
            </Button>
          </Box>
        </CardFooter>
      </Card>
    </Modal>
  );
}
