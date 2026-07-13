import { useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import PasswordField from "~/components/PasswordField";
import api from "~/utils/api/Api";
import { GitDetails } from "./types.d";

interface Props {
  details?: GitDetails;
  closeModal: () => void;
  onSuccess: (message: string) => void;
}

export default function BitbucketModal({
  closeModal,
  onSuccess,
  details,
}: Props) {
  const bb = details?.bitbucket;
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [clientId, setClientId] = useState(bb?.clientId || "");
  const [deployKey, setDeployKey] = useState("");
  const [clientSecret, setClientSecret] = useState("");

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
            provider: "bitbucket",
            clientId,
            clientSecret,
            ...(deployKey.trim() && { deployKey }),
          };

          api
            .post("/admin/git/configure", payload)
            .then(() => {
              onSuccess("Bitbucket configuration saved successfully");
            })
            .catch(res => {
              if (res?.error) {
                setError(res.error);
              } else {
                setError("Something went wrong while configuring Bitbucket");
              }
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title="Configure Bitbucket"
          subtitle="Configure your Bitbucket application to enable authentication and access private repositories"
        />

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Client ID"
            placeholder="Enter Bitbucket Client ID"
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
            label="Client secret"
            placeholder={
              bb?.hasClientSecret
                ? "***************"
                : "The client secret of your Bitbucket application"
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

          <PasswordField
            label="Deploy key"
            placeholder={
              bb?.hasDeployKey
                ? "Enter deploy key"
                : "Enter deploy key (optional)"
            }
            value={deployKey}
            required={bb?.hasDeployKey ? true : false}
            onChange={e => setDeployKey(e.target.value)}
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
            variant="contained"
            color="secondary"
            type="submit"
            disabled={loading}
            sx={{ ml: 2 }}
          >
            Configure
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
