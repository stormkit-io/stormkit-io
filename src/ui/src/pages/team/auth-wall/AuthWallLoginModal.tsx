import type { TeamAuthWallApp, TeamAuthWallLogin } from "./actions";
import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import TextField from "@mui/material/TextField";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import MultiSelect from "~/components/MultiSelect";
import { createTeamLogin, updateTeamLogin } from "./actions";

interface Props {
  team: Team;
  apps: TeamAuthWallApp[];

  // login is the login to edit. When missing, a new login is created.
  login?: TeamAuthWallLogin;
  onClose: () => void;
  onSuccess: () => void;
}

export default function AuthWallLoginModal({
  team,
  apps,
  login,
  onClose,
  onSuccess,
}: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [allApps, setAllApps] = useState(login?.allApps ?? false);
  const [appIds, setAppIds] = useState<string[]>(login?.appIds ?? []);
  const [magicLink, setMagicLink] = useState(login?.magicLink ?? false);
  const [removePassword, setRemovePassword] = useState(false);
  const hasPassword = login?.hasPassword !== false;

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();

    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    if (!login && !data.email) {
      return setError("Email is a required field.");
    }

    const keepsPassword = login
      ? (hasPassword && !removePassword) || !!data.password
      : !!data.password;

    if (!keepsPassword && !magicLink) {
      return setError("Set a password, allow magic links, or both.");
    }

    if (!allApps && appIds.length === 0) {
      return setError("Select at least one app, or all apps.");
    }

    setIsLoading(true);

    const request = login
      ? updateTeamLogin({
          teamId: team.id,
          id: login.id,
          metadata: data.metadata,
          magicLink,
          allApps,
          appIds: allApps ? [] : appIds,
          ...(removePassword
            ? { removePassword: true }
            : data.password
              ? { password: data.password }
              : {}),
        })
      : createTeamLogin({
          teamId: team.id,
          email: data.email,
          password: data.password,
          metadata: data.metadata,
          magicLink,
          allApps,
          appIds: allApps ? [] : appIds,
        });

    request
      .then(() => {
        onSuccess();
        onClose();
      })
      .catch(async res => {
        const body = await res?.json?.().catch(() => undefined);

        setError(body?.error || "Something went wrong while saving the login.");
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <Modal open maxWidth="600px" onClose={onClose}>
      <Card component="form" error={error} onSubmit={submitHandler}>
        <CardHeader
          title={login ? "Edit login" : "New login"}
          subtitle="A login passes the Auth Wall of the apps you choose, in every environment where the wall is on."
        />
        <Box>
          <TextField
            label="Email address"
            variant="filled"
            autoComplete="off"
            defaultValue={login?.email || ""}
            disabled={!!login}
            fullWidth
            name="email"
            autoFocus={!login}
            placeholder="jane@doe.org"
            sx={{ mb: 4 }}
          />
          <TextField
            label="Password"
            type="password"
            variant="filled"
            autoComplete="new-password"
            defaultValue=""
            fullWidth
            name="password"
            disabled={removePassword}
            helperText={
              login && hasPassword
                ? "Leave empty to keep the current password."
                : magicLink
                  ? "Optional with magic links. 8 to 72 characters."
                  : "8 to 72 characters."
            }
            sx={{ mb: 2 }}
          />
          <Box sx={{ mb: 4 }}>
            <FormControlLabel
              label="Allow signing in with a link sent by email"
              control={
                <Checkbox
                  color="secondary"
                  checked={magicLink}
                  onChange={e => {
                    setMagicLink(e.target.checked);

                    if (!e.target.checked) {
                      setRemovePassword(false);
                    }
                  }}
                />
              }
            />
            {login && hasPassword && magicLink && (
              <FormControlLabel
                label="Remove the password, so the login signs in with links only"
                control={
                  <Checkbox
                    color="secondary"
                    checked={removePassword}
                    onChange={e => setRemovePassword(e.target.checked)}
                  />
                }
              />
            )}
          </Box>
          <TextField
            label="Metadata"
            variant="filled"
            autoComplete="off"
            defaultValue={login?.metadata || ""}
            fullWidth
            name="metadata"
            placeholder='{"tenant":"acme"}'
            helperText="Optional. Passed to your app in the X-Auth-Wall-Metadata header. Visible to admins, so don't put secrets here. Changes reach a visitor on their next login."
            slotProps={{ htmlInput: { maxLength: 1024 } }}
            sx={{ mb: 4 }}
          />
          <Box sx={{ mb: 4 }}>
            <FormControlLabel
              sx={{ mb: allApps ? 0 : 2 }}
              label="All apps, including apps created later"
              control={
                <Checkbox
                  color="secondary"
                  checked={allApps}
                  onChange={e => setAllApps(e.target.checked)}
                />
              }
            />
            {!allApps && (
              <MultiSelect
                label="Apps"
                placeholder="Select apps"
                items={apps.map(app => ({
                  value: app.id,
                  text: app.displayName,
                }))}
                selected={appIds}
                onSelect={setAppIds}
              />
            )}
          </Box>
        </Box>
        <CardFooter>
          <Box sx={{ textAlign: "right" }}>
            <Button
              type="submit"
              variant="contained"
              color="secondary"
              loading={isLoading}
            >
              {login ? "Save" : "Create"}
            </Button>
          </Box>
        </CardFooter>
      </Card>
    </Modal>
  );
}
