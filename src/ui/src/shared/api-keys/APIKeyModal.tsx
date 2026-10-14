import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextInput from "@mui/material/TextField";
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import Switch from "@mui/material/Switch";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";

interface APIKeyModalProps {
  onClose: () => void;
  onSubmit: (name: string, readOnly: boolean) => void;
  error?: string;
  loading?: boolean;
  forceReadOnly?: boolean;
}

export default function APIKeyModal({
  onSubmit,
  onClose,
  loading,
  error,
  forceReadOnly,
}: APIKeyModalProps) {
  const [name, setName] = useState<string>("");
  const [readOnly, setReadOnly] = useState(!!forceReadOnly);

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
        <Box sx={{ mb: 4 }}>
          <FormControlLabel
            control={
              <Switch
                color="secondary"
                checked={readOnly}
                disabled={forceReadOnly}
                onChange={e => setReadOnly(e.target.checked)}
              />
            }
            label="Read only"
          />
          <Typography variant="body2" color="text.secondary">
            A read-only key can fetch data but cannot deploy, change or delete
            anything, or reveal environment variable values. On a user-level
            key, MCP clients only see read-only tools.
          </Typography>
        </Box>
        <CardFooter>
          <Button
            loading={loading}
            variant="contained"
            color="secondary"
            onClick={() => onSubmit(name, readOnly)}
          >
            Create
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
