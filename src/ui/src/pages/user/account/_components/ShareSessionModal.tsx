import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import MenuItem from "@mui/material/MenuItem";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CopyBox from "~/components/CopyBox";
import { RootContext } from "~/pages/Root.context";
import { createSharedSession } from "../actions";

const ENTERPRISE_DURATIONS = [1, 3, 6, 12, 24];

interface Props {
  onClose: () => void;
}

export default function ShareSessionModal({ onClose }: Props) {
  const { details } = useContext(RootContext);
  const isEnterprise = details?.license?.edition === "enterprise";

  const [hours, setHours] = useState(1);
  const [token, setToken] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleCreate = () => {
    setLoading(true);
    setError("");

    createSharedSession(hours)
      .then(({ token }) => {
        setToken(token);
      })
      .catch(() => {
        setError(
          "Something went wrong while creating the session. Please try again.",
        );
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Modal open onClose={onClose}>
      <Card error={error} contentPadding={false}>
        <CardHeader
          title="Share session"
          subtitle="Generate a temporary token that can be shared over a private channel to grant access on your behalf."
        />
        {isEnterprise && !token && (
          <Box sx={{ px: 4, pb: 2 }}>
            <TextField
              select
              fullWidth
              label="Session duration"
              value={hours}
              onChange={e => setHours(Number(e.target.value))}
              variant="filled"
            >
              {ENTERPRISE_DURATIONS.map(h => (
                <MenuItem key={h} value={h}>
                  {h}h
                </MenuItem>
              ))}
            </TextField>
          </Box>
        )}
        {token ? (
          <Box sx={{ px: 4, pb: 4 }}>
            <Typography sx={{ mb: 2, color: "text.secondary" }}>
              Copy this snippet and run it in the browser console of the target
              instance. This token expires in {hours}h:
            </Typography>
            <CopyBox
              value={`localStorage.setItem('skit_token', JSON.stringify('${token}'))`}
            />
          </Box>
        ) : (
          <CardFooter>
            <Button variant="outlined" onClick={onClose} sx={{ mr: 2 }}>
              Cancel
            </Button>
            <Button
              variant="contained"
              color="secondary"
              onClick={handleCreate}
              loading={loading}
            >
              Create session
            </Button>
          </CardFooter>
        )}
      </Card>
    </Modal>
  );
}
