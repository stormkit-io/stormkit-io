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

export default function ForgejoModal({ closeModal, onSuccess, details }: Props) {
  const fj = details?.forgejo;
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [baseUrl, setBaseUrl] = useState(fj?.baseUrl || "");
  const [clientId, setClientId] = useState(fj?.clientId || "");
  const [clientSecret, setClientSecret] = useState("");
  const [redirectUrl, setRedirectUrl] = useState(fj?.redirectUrl || "");

  return (
    <Modal open onClose={closeModal} maxWidth="md">
      <Card
        component="form"
        error={error}
        onSubmit={e => {
          e.preventDefault();

          if (!baseUrl.trim()) {
            setError("Forgejo URL is required");
            return;
          }

          if (!clientId.trim()) {
            setError("Client ID is required");
            return;
          }

          // A blank secret keeps the stored one.
          if (!clientSecret.trim() && !fj?.hasClientSecret) {
            setError("Client Secret is required");
            return;
          }

          setLoading(true);

          api
            .post("/admin/git/configure", {
              provider: "forgejo",
              baseUrl,
              clientId,
              clientSecret,
              redirectUrl,
            })
            .then(() => {
              onSuccess("Forgejo configuration saved successfully");
            })
            .catch(async (res: Response) => {
              const data = await res.json().catch(() => undefined);

              setError(
                data?.error || "Something went wrong while configuring Forgejo"
              );
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title="Configure Forgejo"
          subtitle="Connect your Forgejo instance to enable authentication and access private repositories"
        />

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Forgejo URL"
            placeholder="https://git.example.org"
            value={baseUrl}
            onChange={e => setBaseUrl(e.target.value)}
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
            label="Client ID"
            placeholder="Enter the Client ID of your Forgejo OAuth2 application"
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
              fj?.hasClientSecret
                ? "***************"
                : "The client secret of your Forgejo OAuth2 application"
            }
            type="password"
            value={clientSecret}
            onChange={e => setClientSecret(e.target.value)}
            fullWidth
            required={!fj?.hasClientSecret}
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
            placeholder="The URL to redirect for oAuth callbacks. Leave empty for auto-detection."
            value={redirectUrl}
            onChange={e => setRedirectUrl(e.target.value)}
            fullWidth
            variant="filled"
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
