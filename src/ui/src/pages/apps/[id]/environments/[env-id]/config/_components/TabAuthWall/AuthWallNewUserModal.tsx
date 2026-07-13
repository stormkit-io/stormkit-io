import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { createNewLogin } from "./actions";

interface Props {
  appId: string;
  envId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function AuthWallNewUserModal({
  onClose,
  onSuccess,
  appId,
  envId,
}: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    if (!data.email || !data.password) {
      return setError("Email and password are required fields.");
    }

    setIsLoading(true);

    createNewLogin({ appId, envId, email: data.email, password: data.password })
      .then(() => {
        onClose();
        onSuccess();
      })
      .catch(async e => {
        const data = await e.json();

        setError(
          data.error || "Something went wrong while creating the new login."
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <Modal open maxWidth="500px" onClose={onClose}>
      <Card component="form" error={error} onSubmit={submitHandler}>
        <CardHeader
          title="Create new login"
          subtitle="Logins are used to authenticate for protected deployments."
        />
        <Box>
          <TextField
            label="Email address"
            variant="filled"
            autoComplete="off"
            defaultValue={""}
            fullWidth
            name="email"
            autoFocus
            placeholder="jane@doe.org"
            sx={{ mb: 4 }}
          />
          <TextField
            label="Password"
            type="password"
            variant="filled"
            autoComplete="off"
            defaultValue={""}
            fullWidth
            name="password"
            sx={{ mb: 4 }}
          />
        </Box>
        <CardFooter>
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
        </CardFooter>
      </Card>
    </Modal>
  );
}
