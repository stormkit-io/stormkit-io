import type { TeamAuthWallApp, TeamAuthWallLogin } from "./actions";
import { FormEventHandler, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import MultiSelect from "~/components/MultiSelect";
import PasswordField from "~/components/PasswordField";
import { createTeamLogin, updateTeamLogin } from "./actions";

interface Props {
  team: Team;
  apps: TeamAuthWallApp[];

  // login is the login to edit. When missing, a new login is created.
  login?: TeamAuthWallLogin;

  magicLinkAvailable?: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function AuthWallLoginModal({
  team,
  apps,
  login,
  magicLinkAvailable = true,
  onClose,
  onSuccess,
}: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [allApps, setAllApps] = useState(login?.allApps ?? false);
  const [appIds, setAppIds] = useState<string[]>(login?.appIds ?? []);
  const [magicLink, setMagicLink] = useState(login?.magicLink ?? false);
  const hasPassword = !!login && login.hasPassword !== false;
  const [password, setPassword] = useState(login ? hasPassword : true);

  const canEnableMagicLink = magicLinkAvailable || !!login?.magicLink;

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

    if (!password && !magicLink) {
      return setError("Turn on a password, magic links, or both.");
    }

    if (password && !hasPassword && !data.password) {
      return setError("Enter a password, or turn passwords off.");
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
          ...(!password && hasPassword
            ? { removePassword: true }
            : password && data.password
              ? { password: data.password }
              : {}),
        })
      : createTeamLogin({
          teamId: team.id,
          email: data.email,
          password: password ? data.password : "",
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
    <Modal open maxWidth="640px" onClose={onClose}>
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
          <SwitchOption
            label="Sign in with a password"
            description="The visitor enters their email and a password on the login page."
            checked={password}
            onChange={setPassword}
          >
            <PasswordField
              label="Password"
              variant="filled"
              autoComplete="new-password"
              defaultValue=""
              name="password"
              helperText={
                hasPassword
                  ? "Leave empty to keep the current password."
                  : "8 to 72 characters."
              }
            />
          </SwitchOption>
          <SwitchOption
            label="Sign in with a link sent by email"
            description={
              canEnableMagicLink
                ? "The visitor gets a one-time sign-in link by email. Links expire after 15 minutes."
                : "Configure a mailer under Admin > Mailer to use magic links."
            }
            disabled={!canEnableMagicLink}
            checked={magicLink}
            onChange={setMagicLink}
          />
          <SwitchOption
            label="All apps, including apps created later"
            description="Turn off to choose the apps this login opens."
            checked={allApps}
            onChange={setAllApps}
          />
          {!allApps && (
            <Box sx={{ mb: 4 }}>
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
            </Box>
          )}
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
        </Box>
        <CardFooter>
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            loading={isLoading}
            sx={{ ml: 2 }}
          >
            {login ? "Save" : "Create"}
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}

interface SwitchOptionProps {
  label: string;
  description: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;

  // children are shown below the option while it is on.
  children?: React.ReactNode;
}

function SwitchOption({
  label,
  description,
  checked,
  disabled,
  onChange,
  children,
}: SwitchOptionProps) {
  return (
    <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
      <FormControlLabel
        sx={{ pl: 0, ml: 0 }}
        label={label}
        disabled={disabled}
        control={
          <Switch
            color="secondary"
            checked={checked}
            onChange={e => onChange(e.target.checked)}
          />
        }
        labelPlacement="start"
      />
      <Typography sx={{ color: "text.secondary" }}>{description}</Typography>
      {checked && children && <Box sx={{ mt: 2 }}>{children}</Box>}
    </Box>
  );
}
