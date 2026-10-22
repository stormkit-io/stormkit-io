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

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();

    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    if (!login && (!data.email || !data.password)) {
      return setError("Email and password are required fields.");
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
          allApps,
          appIds: allApps ? [] : appIds,
          ...(data.password ? { password: data.password } : {}),
        })
      : createTeamLogin({
          teamId: team.id,
          email: data.email,
          password: data.password,
          metadata: data.metadata,
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
            helperText={
              login
                ? "Leave empty to keep the current password."
                : "8 to 72 characters."
            }
            sx={{ mb: 4 }}
          />
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
