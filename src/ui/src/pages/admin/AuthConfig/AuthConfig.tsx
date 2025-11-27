import type { SignUpMode, AuthConfig } from "./actions";
import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import TextField from "@mui/material/TextField";
import Select from "@mui/material/Select";
import Option from "@mui/material/MenuItem";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import PendingUsers from "./PendingUsers";
import { useFetchAuthConfig, updateAuthConfig } from "./actions";

interface ConfigProps {
  config?: AuthConfig;
  error?: string;
  loading?: boolean;
}

function Config({ config, loading, error }: ConfigProps) {
  const [signUpMode, setSignUpMode] = useState<SignUpMode>("on");
  const [whitelist, setWhitelist] = useState<string>();
  const [whitelistError, setWhitelistError] = useState(false);
  const [updateError, setUpdateError] = useState<string>();
  const [updateLoading, setUpdateLoading] = useState(false);
  const [updateSuccess, setUpdateSuccess] = useState<string>();

  useEffect(() => {
    if (config?.signUpMode) {
      setSignUpMode(config.signUpMode);
    }

    if (config?.whitelist) {
      setWhitelist(config.whitelist.join(", "));
    }
  }, [config]);

  return (
    <Card
      loading={loading}
      error={error || updateError}
      success={updateSuccess}
      sx={{ backgroundColor: "container.transparent" }}
      contentPadding={false}
      component="form"
      onSubmit={e => {
        e.preventDefault();

        setWhitelistError(false);

        const wh: string[] = [];
        const expectNegation: boolean = whitelist?.[0]?.[0] === "!";
        let hasError = false;

        if (whitelist && whitelist.trim() !== "") {
          whitelist.split(",").forEach(i => {
            const trimmed = i.trim();

            // If the first item has negation, remaining items
            // should also have negation.
            if (expectNegation && trimmed[0] !== "!") {
              hasError = true;
            } else if (!expectNegation && trimmed[0] === "!") {
              hasError = true;
            }

            wh.push(trimmed);
          });
        }

        if (hasError) {
          setWhitelistError(hasError);
          return;
        }

        setUpdateLoading(true);

        updateAuthConfig({
          whitelist: wh,
          signUpMode,
        })
          .then(() => {
            setUpdateError(undefined);
            setUpdateSuccess(
              "The user management configuration is updated successfully. Changes will take effect for new users."
            );
          })
          .catch(() => {
            setUpdateSuccess(undefined);
            setUpdateError(
              "Something went wrong while updating user management config."
            );
          })
          .finally(() => {
            setUpdateLoading(false);
          });
      }}
    >
      <CardHeader
        title="User management"
        subtitle="Configure how your users can access Stormkit"
      />
      <Box sx={{ px: 4 }}>
        <FormControl variant="standard" fullWidth>
          <InputLabel id="auth-config-label" sx={{ pl: 2, pt: 1.25 }}>
            Sign up mode
          </InputLabel>
          <Select
            labelId="auth-config-label"
            name="signUpMode"
            variant="filled"
            value={signUpMode}
            fullWidth
            onChange={e => {
              setSignUpMode(e.target.value as SignUpMode);
            }}
          >
            <Option value="off">Off (no new users allowed)</Option>
            <Option value="on">On (all users are allowed)</Option>
            <Option value="waitlist">Approval mode</Option>
          </Select>
        </FormControl>
      </Box>
      {signUpMode === "waitlist" && (
        <Box sx={{ p: 4, pb: 2 }}>
          <TextField
            variant="filled"
            name="whitelist"
            label="Whitelist"
            value={whitelist}
            error={whitelistError}
            fullWidth
            autoFocus
            autoComplete="off"
            onChange={e => setWhitelist(e.target.value)}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
            helperText={
              <Typography component="span" sx={{ mb: 2, display: "block" }}>
                {whitelistError
                  ? `All domains must either be allowed or denied. You cannot mix negated domains (!) with regular domains`
                  : `Specify which email domains are automatically approved (e.g. example.org). Use ! prefix to deny specific domains (e.g. !spam.com)`}
              </Typography>
            }
          />
        </Box>
      )}
      <CardFooter>
        <Button
          variant="contained"
          color="secondary"
          type="submit"
          loading={updateLoading}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}

export default function AuthConfig() {
  const { error, loading, config } = useFetchAuthConfig();

  return (
    <Box>
      <Config error={error} loading={loading} config={config} />
      <PendingUsers />
    </Box>
  );
}
