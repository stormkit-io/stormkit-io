import type { SxProps } from "@mui/material";
import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";
import DeleteIcon from "@mui/icons-material/Delete";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import CopyBox from "~/components/CopyBox/CopyBox";
import APIKeyModal from "./APIKeyModal";
import * as actions from "./actions";

const { useFetchAPIKeys, generateNewAPIKey, deleteAPIKey } = actions;

interface Props {
  cardId?: string;
  cardSx?: SxProps;
  title?: string;
  subtitle: string;
  emptyMessage: string;
  apiKeyProps: {
    appId?: string;
    envId?: string;
    teamId?: string;
    userId?: string;
    scope: string;
  };
}

export default function APIKeyList({
  cardId,
  cardSx,
  title = "API Keys",
  subtitle,
  emptyMessage,
  apiKeyProps,
}: Props) {
  const [refreshToken, setRefreshToken] = useState<number>();
  const [modalError, setModalError] = useState("");
  const [modalLoading, setModalLoading] = useState(false);
  const [apiKeyToDelete, setApiKeyToDelete] = useState<APIKey>();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newlyCreatedToken, setNewlyCreatedToken] = useState<string>();
  const { loading, error, keys, setKeys } = useFetchAPIKeys({
    ...apiKeyProps,
    refreshToken,
  });

  const handleNewKey = (name: string) => {
    setModalLoading(true);

    generateNewAPIKey({ ...apiKeyProps, name })
      .then(apiKey => {
        setKeys([...keys, apiKey]);
        setNewlyCreatedToken(apiKey.token);
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
      id={cardId}
      loading={loading}
      sx={cardSx}
      info={!loading && keys.length === 0 ? emptyMessage : ""}
      error={
        error
          ? "An error occurred while fetching your API key. Please try again later."
          : ""
      }
    >
      <CardHeader title={title} subtitle={subtitle} />

      {newlyCreatedToken && (
        <Alert
          severity="info"
          sx={{ mx: 2, mb: 2 }}
          onClose={() => setNewlyCreatedToken(undefined)}
        >
          <Typography variant="body2" sx={{ mb: 1 }}>
            Make sure to copy your new API key now. It won't be shown again.
          </Typography>
          <CopyBox sx={{ mt: 1 }} value={newlyCreatedToken} />
        </Alert>
      )}

      <Box>
        {keys.map(apiKey => (
          <CardRow
            key={apiKey.id}
            menuLabel={`expand-${apiKey.id}`}
            menuItems={[
              {
                text: "Delete",
                icon: <DeleteIcon />,
                onClick: () => {
                  setApiKeyToDelete(apiKey);
                },
              },
            ]}
          >
            <Typography>{apiKey.name}</Typography>
          </CardRow>
        ))}
      </Box>

      <CardFooter>
        <Button
          type="button"
          variant="contained"
          color="secondary"
          onClick={() => setIsModalOpen(true)}
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
          onCancel={() => setApiKeyToDelete(undefined)}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            deleteAPIKey(apiKeyToDelete)
              .then(() => {
                setApiKeyToDelete(undefined);
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setError("Something went wrong while deleting the API key.");
              })
              .finally(() => {
                setLoading(false);
              });

            return "";
          }}
        >
          This will delete the API key. If you have any integration that uses
          this key, it will stop working.
        </ConfirmModal>
      )}
    </Card>
  );
}
