import type { FormEventHandler } from "react";
import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Select from "@mui/material/Select";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CopyBox from "~/components/CopyBox";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { inviteMember } from "./actions";
import Typography from "@mui/material/Typography";

interface Props {
  teamId: string;
  onClose: () => void;
}

export default function InviteMemberModal({ teamId, onClose }: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [token, setToken] = useState("");

  const handleSubmit: FormEventHandler<HTMLElement> = e => {
    e.preventDefault();

    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    setIsLoading(false);
    setError("");
    setToken("");

    inviteMember({ teamId, email: data.email, role: data.role as TeamRole })
      .then(({ token }) => {
        setToken(token);
      })
      .catch(async res => {
        if (res.status === 401) {
          setError("You are not authorized to invite team members.");
        } else {
          const { error } = await res.json();
          setError(error || "Something went wrong while inviting user.");
        }
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <Modal open onClose={onClose}>
      <Card
        component="form"
        error={error}
        success={
          token ? (
            <Box>
              <Typography sx={{ mb: 2 }}>
                Share the following link with your colleague to invite them to
                the team.
              </Typography>
              <CopyBox
                fullWidth
                value={
                  new URL(`/invitation/accept?token=${token}`, document.baseURI)
                    .href
                }
              />
            </Box>
          ) : undefined
        }
        onSubmit={handleSubmit}
      >
        <CardHeader
          title="Invite team member"
          subtitle="Team members will be able to see and modify all projects in the team."
        />
        <Box sx={{ display: "flex" }}>
          <TextField
            label="Email"
            variant="filled"
            autoComplete="off"
            defaultValue={""}
            fullWidth
            name="email"
            autoFocus
            sx={{ mb: 4, mr: 2 }}
          />
          <FormControl variant="standard" fullWidth>
            <InputLabel id="team-member-role" sx={{ pl: 2, pt: 1 }}>
              Role
            </InputLabel>
            <Select
              labelId="team-member-role"
              variant="filled"
              name="role"
              defaultValue="admin"
              sx={{ minWidth: 150 }}
            >
              <MenuItem value="admin">Admin</MenuItem>
              <MenuItem value="developer">Developer</MenuItem>
            </Select>
          </FormControl>
        </Box>
        <CardFooter>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={isLoading}
          >
            Invite
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
