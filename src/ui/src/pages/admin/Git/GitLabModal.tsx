import { useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";
import { GitDetails } from "./types.d";

interface Props {
  closeModal: () => void;
  onSuccess: (message: string) => void;
  details?: GitDetails;
}

export default function GitLabModal({ closeModal, onSuccess, details }: Props) {
  const gl = details?.gitlab;
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [clientId, setClientId] = useState(gl?.clientId || "");
  const [clientSecret, setClientSecret] = useState("");
  const [redirectUrl, setRedirectUrl] = useState(gl?.redirectUrl || "");

  return (
    <Modal open onClose={closeModal} maxWidth="md">
      <Card
        component="form"
        error={error}
        onSubmit={e => {
          e.preventDefault();

          if (!clientId.trim()) {
            setError("Client ID is required");
            return;
          }

          if (!clientSecret.trim()) {
            setError("Client Secret is required");
            return;
          }

          setLoading(true);

          const payload = {
            provider: "gitlab",
            clientId,
            clientSecret,
            redirectUrl,
          };

          api
            .post("/admin/git/configure", payload)
            .then(() => {
              onSuccess("GitLab configuration saved successfully");
            })
            .catch(res => {
              if (res?.error) {
                setError(res.error);
              } else {
                setError("Something went wrong while configuring GitLab");
              }
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title="Configure GitLab"
          subtitle="Configure your GitLab application to enable authentication and access private repositories"
        />

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Client ID"
            placeholder="Enter GitLab Client ID"
            value={clientId}
            onChange={e => setClientId(e.target.value)}
            fullWidth
            required
            variant="filled"
            sx={{ mb: 3 }}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />

          <TextField
            label="Client Secret"
            placeholder={
              gl?.hasClientSecret
                ? "***************"
                : "The client secret of your GitLab application"
            }
            type="password"
            value={clientSecret}
            onChange={e => setClientSecret(e.target.value)}
            fullWidth
            required
            variant="filled"
            sx={{ mb: 3 }}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />

          <TextField
            label="Redirect URL"
            placeholder={
              "The URL to redirect for oAuth callbacks. Leave empty for auto-detection."
            }
            value={redirectUrl}
            onChange={e => setRedirectUrl(e.target.value)}
            fullWidth
            required
            variant="filled"
            sx={{ mb: 3 }}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />
        </Box>

        <CardFooter>
          <Button onClick={closeModal} disabled={loading}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            disabled={loading}
            loading={loading}
            sx={{ ml: 2 }}
          >
            Configure
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
