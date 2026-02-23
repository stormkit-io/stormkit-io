import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import VisibilityIcon from "@mui/icons-material/Visibility";
import VisibilityOffIcon from "@mui/icons-material/VisibilityOff";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import Snackbar from "@mui/material/Snackbar";
import DeleteIcon from "@mui/icons-material/Delete";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import APIKeyModal from "~/shared/api-keys/APIKeyModal";
import * as actions from "~/shared/api-keys/actions";

const { useFetchAPIKeys, generateNewAPIKey, deleteAPIKey } = actions;

interface Props {
  app: App;
  environment: Environment;
}

export default function TabAPIKey({ app, environment: env }: Props) {
  const [clicked, setClicked] = useState<string>();
  const [isVisible, setIsVisible] = useState<string>("");
  const [refreshToken, setRefreshToken] = useState<number>();
  const [success, setSuccess] = useState(false);
  const [modalError, setModalError] = useState("");
  const [modalLoading, setModalLoading] = useState(false);
  const [apiKeyToDelete, setApiKeyToDelete] = useState<APIKey>();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const { loading, error, keys, setKeys } = useFetchAPIKeys({
    appId: app.id,
    envId: env.id!,
    refreshToken,
  });

  const handleNewKey = (name: string) => {
    setModalLoading(true);
    setSuccess(false);

    generateNewAPIKey({
      appId: app.id,
      envId: env.id!,
      name,
      scope: "env",
    })
      .then(apiKey => {
        setKeys([...keys, apiKey]);
        setSuccess(true);
        setIsModalOpen(false);
      })
      .catch(async res => {
        if (res.status === 400) {
          const { error } = (await res.json()) as { error: string };
          setModalError(error);
        }
      })
      .finally(() => {
        setModalLoading(false);
      });
  };

  return (
    <Card
      id="api-keys"
      loading={loading}
      info={
        !loading && keys.length === 0
          ? "You do not have an API key associated with this environment."
          : ""
      }
      error={
        error
          ? "An error occurred while fetching your API key. Please try again later."
          : ""
      }
      success={success ? "Your API key has been successfully updated." : ""}
    >
      <CardHeader
        title="API Keys"
        subtitle="This key will allow you to interact with our API and modify this environment."
      />

      <Box>
        {keys.map(apiKey => (
          <CardRow
            key={apiKey.token}
            menuLabel={`expand-${apiKey.id}`}
            menuItems={[
              {
                text: "Copy to Clipboard",
                icon: <ContentCopyIcon />,
                onClick: () => {
                  setClicked(apiKey.id);
                  navigator.clipboard.writeText(apiKey.token);
                },
              },
              {
                text: "Toggle visibility",
                icon:
                  isVisible === apiKey.id ? (
                    <VisibilityIcon />
                  ) : (
                    <VisibilityOffIcon />
                  ),
                onClick: () => {
                  setIsVisible(isVisible === apiKey.id ? "" : apiKey.id);
                },
              },
              {
                text: "Delete",
                icon: <DeleteIcon />,
                onClick: () => {
                  setApiKeyToDelete(apiKey);
                },
              },
            ]}
          >
            <Box
              sx={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
              }}
            >
              <Box>
                <Typography>{apiKey.name}</Typography>
                <Typography
                  sx={{
                    textOverflow: "ellipsis",
                    overflow: "hidden",
                    maxWidth: { md: "300px", lg: "none" },
                  }}
                >
                  {isVisible === apiKey.id ? apiKey.token : "*".repeat(32)}
                </Typography>
              </Box>
            </Box>
          </CardRow>
        ))}
      </Box>

      <CardFooter>
        <Button
          type="button"
          variant="contained"
          color="secondary"
          onClick={() => {
            setIsModalOpen(true);
          }}
        >
          New API Key
        </Button>
      </CardFooter>

      {isModalOpen && (
        <APIKeyModal
          error={modalError}
          loading={modalLoading}
          onClose={() => setIsModalOpen(false)}
          onSubmit={handleNewKey}
        />
      )}
      {apiKeyToDelete && (
        <ConfirmModal
          onCancel={() => {
            setApiKeyToDelete(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);
            setSuccess(false);

            deleteAPIKey(apiKeyToDelete)
              .then(() => {
                setRefreshToken(Date.now());
              })
              .catch(e => {
                console.log(e);
                setError("Something went wrong while deleting the API key.");
              })
              .finally(() => {
                setLoading(false);
                setApiKeyToDelete(undefined);
              });

            return "";
          }}
        >
          This will delete the API key. If you have any integration that uses
          this key, it will stop working.
        </ConfirmModal>
      )}
      {clicked && (
        <Snackbar
          color="success"
          open
          autoHideDuration={5000}
          onClose={() => {
            setClicked(undefined);
          }}
          message={
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <CheckCircleIcon color="success" sx={{ mr: 1 }} />
              API Key copied to clipboard successfully
            </Box>
          }
        />
      )}
    </Card>
  );
}
