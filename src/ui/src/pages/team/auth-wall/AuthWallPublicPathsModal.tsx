import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { updateEnvAuthWall } from "./actions";

interface Props {
  team: Team;
  envId: string;

  // name names the environment in the title, e.g. "my-app / production".
  name: string;
  publicPaths: string[];
  onClose: () => void;
  onSuccess: (publicPaths: string[]) => void;
}

export default function AuthWallPublicPathsModal({
  team,
  envId,
  name,
  publicPaths,
  onClose,
  onSuccess,
}: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();

    const form = e.target as HTMLFormElement;
    const value = new FormData(form).get("publicPaths")?.toString() || "";
    const paths = value
      .split("\n")
      .map(path => path.trim())
      .filter(Boolean);

    setIsLoading(true);
    setError("");

    updateEnvAuthWall({ teamId: team.id, envId, publicPaths: paths })
      .then(() => {
        onSuccess(paths);
        onClose();
      })
      .catch(async res => {
        const body = await res?.json?.().catch(() => undefined);

        setError(
          body?.error || "Something went wrong while saving the public paths.",
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <Modal open maxWidth="600px" onClose={onClose}>
      <Card component="form" error={error} onSubmit={submitHandler}>
        <CardHeader
          title={`Public paths of ${name}`}
          subtitle="These paths skip the Auth Wall and are open to everyone. Use them for callers that can't log in, such as webhooks."
        />
        <Box>
          <TextField
            label="Public paths"
            variant="filled"
            autoComplete="off"
            defaultValue={publicPaths.join("\n")}
            fullWidth
            multiline
            minRows={4}
            autoFocus
            name="publicPaths"
            placeholder={"/api/webhooks/*\n/robots.txt"}
            helperText="One path per line. End a path with /* to include everything below it. Your app must check who is calling these paths."
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
