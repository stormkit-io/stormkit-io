import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextInput from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";

interface APIKeyModalProps {
  onClose: () => void;
  onSubmit: (name: string) => void;
  error?: string;
  loading?: boolean;
}

export default function APIKeyModal({
  onSubmit,
  onClose,
  loading,
  error,
}: APIKeyModalProps) {
  const [name, setName] = useState<string>("");

  return (
    <Modal open onClose={onClose}>
      <Card error={error}>
        <CardHeader title="Generate New Key" />
        <Box sx={{ mb: 4 }}>
          <TextInput
            variant="filled"
            label="API Key name"
            placeholder="e.g. Default"
            fullWidth
            autoFocus
            autoComplete="off"
            value={name}
            onChange={e => setName(e.target.value)}
            helperText={
              <Typography component="span" sx={{ mt: 1 }}>
                The name will be used in the UI. It helps distinguishing your
                API keys.
              </Typography>
            }
          />
        </Box>
        <CardFooter>
          <Button
            loading={loading}
            variant="contained"
            color="secondary"
            onClick={() => onSubmit(name)}
          >
            Create
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
