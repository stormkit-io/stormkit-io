import React, { useState } from "react";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";

type ConfirmModalCallback = ({
  setLoading,
  setError,
}: {
  setLoading: (value: boolean) => void;
  setError: (err: string | null) => void;
}) => void;

interface Props {
  title?: string;
  children?: React.ReactNode;
  confirmText?: string;
  onCancel: () => void;
  onConfirm: ConfirmModalCallback;
  typeConfirmationText?: string;
}

const ConfirmModal: React.FC<Props> = ({
  children,
  confirmText,
  typeConfirmationText,
  onConfirm,
  onCancel,
  title = "Confirm action",
}) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmButtonEnabled, setConfirmButtonEnabled] = useState<boolean>(
    !typeConfirmationText
  );

  const handleSuccess = () => {
    if (onConfirm) {
      onConfirm({ setLoading, setError });
    }
  };

  const verifyConfirmationInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.value === typeConfirmationText) {
      setConfirmButtonEnabled(true);
    }
  };

  return (
    <Modal open onClose={onCancel} maxWidth="md">
      <Card error={error}>
        <CardHeader title={title} />
        <Typography component="div" fontSize="medium" sx={{ mb: 0 }}>
          {children}
        </Typography>
        {!typeConfirmationText && (
          <Typography sx={{ mt: 2, mb: 4 }} fontSize="medium">
            Are you sure you want to continue?
          </Typography>
        )}
        {typeConfirmationText && (
          <>
            <Typography fontSize="medium" sx={{ mt: 2 }}>
              Type{" "}
              <Typography
                component="b"
                color="text.secondary"
                fontSize="medium"
              >
                {typeConfirmationText}
              </Typography>{" "}
              in order to proceed.
            </Typography>
            <TextField
              label="Confirmation"
              type="text"
              variant="filled"
              autoComplete="off"
              placeholder={typeConfirmationText}
              onChange={verifyConfirmationInput}
              fullWidth
              autoFocus
              sx={{ my: 4 }}
            />
          </>
        )}
        <CardFooter>
          <Button
            color="info"
            variant="text"
            type="button"
            sx={{ mr: 2 }}
            onClick={onCancel}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            color="secondary"
            loading={loading}
            disabled={!confirmButtonEnabled}
            sx={{ opacity: confirmButtonEnabled ? 1 : 0.5 }}
            onClick={handleSuccess}
          >
            {confirmText || "Yes, continue"}
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
};

export default ConfirmModal;
