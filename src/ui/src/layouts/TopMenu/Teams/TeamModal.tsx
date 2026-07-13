import { useState } from "react";
import { useNavigate } from "react-router";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import { createTeam } from "./actions";

interface Props {
  reload?: () => void;
  onClose: () => void;
}

export default function TeamModal({ onClose, reload }: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  return (
    <Modal open maxWidth="500px" onClose={onClose}>
      <Box sx={{ p: 4 }}>
        <Box sx={{ mb: 4, display: "flex", alignItems: "flex-start" }}>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h6">New Team</Typography>
            <Typography variant="subtitle2" sx={{ opacity: 0.5 }}>
              Use teams to collaborate with other team members.
            </Typography>
          </Box>
        </Box>
        <Box
          component="form"
          onSubmit={e => {
            e.preventDefault();
            const form = e.target as HTMLFormElement;
            const data = Object.fromEntries(
              new FormData(form).entries()
            ) as Record<string, string>;

            if (!data.name) {
              return setError("Team name is a required field.");
            }

            setIsLoading(true);

            createTeam({ name: data.name })
              .then(t => {
                reload?.();
                onClose();
                navigate(`/${t.slug}`, { replace: true });
              })
              .catch(() => {
                setError("Something went wrong while saving the team.");
              })
              .finally(() => {
                setIsLoading(false);
              });
          }}
        >
          <TextField
            label="Team name"
            variant="filled"
            autoComplete="off"
            defaultValue={""}
            fullWidth
            name="name"
            autoFocus
            placeholder="My Awesome Team"
            sx={{ mb: 4 }}
          />
          {error && (
            <Alert color="error" sx={{ mb: 4 }}>
              <AlertTitle>Error</AlertTitle>
              <Typography>{error}</Typography>
            </Alert>
          )}
          <Box sx={{ textAlign: "right" }}>
            <Button
              type="submit"
              variant="contained"
              color="secondary"
              loading={isLoading}
            >
              Create
            </Button>
          </Box>
        </Box>
      </Box>
    </Modal>
  );
}
