import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import VisibilityIcon from "@mui/icons-material/Visibility";
import VisibilityOffIcon from "@mui/icons-material/VisibilityOff";
import DeleteIcon from "@mui/icons-material/Delete";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import Spinner from "~/components/Spinner";
import APIKeyModal from "~/shared/api-keys/APIKeyModal";
import * as actions from "~/shared/api-keys/actions";

const { useFetchAPIKeys, generateNewAPIKey, deleteAPIKey } = actions;

interface Props {
  team: Team;
}

export default function TeamAPIKeys({ team }: Props) {
  const [isVisible, setIsVisible] = useState<string>("");
  const [refreshToken, setRefreshToken] = useState<number>();
  const [success, setSuccess] = useState(false);
  const [modalError, setModalError] = useState("");
  const [modalLoading, setModalLoading] = useState(false);
  const [apiKeyToDelete, setApiKeyToDelete] = useState<APIKey>();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const { loading, error, keys, setKeys } = useFetchAPIKeys({
    teamId: team.id,
    refreshToken,
  });

  const handleNewKey = (name: string) => {
    setModalLoading(true);
    setSuccess(false);

    generateNewAPIKey({
      teamId: team.id,
      name,
      scope: "team",
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
      sx={{ mb: 2 }}
      error={
        error
          ? "An error occurred while fetching your API key. Please try again later."
          : undefined
      }
      success={
        success ? "Your API key has been successfully updated." : undefined
      }
      info={
        !keys.length
          ? "You do not have an API key associated with this team yet."
          : undefined
      }
    >
      <CardHeader
        title="API Keys"
        subtitle="This key will allow you to interact with our API and modify all apps under this team."
      />
      {loading && (
        <Box
          sx={{
            p: 2,
            display: "flex",
            justifyContent: "center",
          }}
        >
          <Spinner />
        </Box>
      )}
      <Box>
        {keys.map(apiKey => (
          <CardRow
            key={apiKey.token}
            menuLabel={`${apiKey.name} menu`}
            menuItems={[
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
              .catch(() => {
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
    </Card>
  );
}
