import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { AuthUser, updateAuthUser } from "./actions";

interface Props {
  envId: string;
  user: AuthUser;
  onClose: () => void;
  onSuccess: () => void;
}

export default function EditUserModal({
  envId,
  user,
  onClose,
  onSuccess,
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

    if (!data.email) {
      return setError("Email is a required field.");
    }

    setIsLoading(true);

    updateAuthUser({
      envId,
      userId: user.id,
      email: data.email,
      firstName: data.firstName || "",
      lastName: data.lastName || "",
    })
      .then(() => {
        onClose();
        onSuccess();
      })
      .catch(async e => {
        const data = await e.json().catch(() => ({}));

        setError(
          data.errors?.[0] ||
            data.error ||
            "Something went wrong while updating the user.",
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
          title="Edit user"
          subtitle="Update the registered user's email and name."
        />
        <Box>
          <TextField
            label="Email address"
            variant="filled"
            autoComplete="off"
            defaultValue={user.email}
            fullWidth
            name="email"
            autoFocus
            placeholder="jane@doe.org"
            sx={{ mb: 4 }}
          />
          <TextField
            label="First name"
            variant="filled"
            autoComplete="off"
            defaultValue={user.firstName}
            fullWidth
            name="firstName"
            sx={{ mb: 4 }}
          />
          <TextField
            label="Last name"
            variant="filled"
            autoComplete="off"
            defaultValue={user.lastName}
            fullWidth
            name="lastName"
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
              Save
            </Button>
          </Box>
        </CardFooter>
      </Card>
    </Modal>
  );
}
