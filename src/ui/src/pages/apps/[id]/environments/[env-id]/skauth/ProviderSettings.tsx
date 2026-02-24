import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Drawer from "@mui/material/Drawer";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import { Switch } from "~/components/Form";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CopyBox from "~/components/CopyBox";
import Api from "~/utils/api/Api";
import type { AuthProvider } from "./actions";

interface Props {
  envId: string;
  isDrawerOpen: boolean;
  provider: AuthProvider;
  onClose: () => void;
  setRefreshToken: (value: number) => void;
}

export default function ProviderSettings({
  isDrawerOpen,
  provider,
  envId,
  onClose,
  setRefreshToken,
}: Props) {
  const [isEnabled, setIsEnabled] = useState(!!provider.enabled);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);

  return (
    <Drawer
      anchor="right"
      open={isDrawerOpen}
      onClose={onClose}
      sx={{ zIndex: 1600 }}
    >
      <Card
        component="form"
        error={error}
        sx={{ minHeight: "100vh", minWidth: "600px", maxWidth: "800px" }}
        onSubmit={e => {
          e.preventDefault();
          const form = e.target as HTMLFormElement;
          const data = Object.fromEntries(
            new FormData(form).entries(),
          ) as Record<string, string>;

          setLoading(true);
          setError(undefined);

          Api.post(`/skauth`, {
            envId,
            providerName: provider.id,
            clientId: data.clientId,
            clientSecret: data.clientSecret,
            status: isEnabled,
          })
            .then(() => {
              setRefreshToken(Date.now());
              onClose();
            })
            .catch(() => {
              setError("Something went wrong while saving provider settings.");
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title={provider.drawerTitle}
          subtitle={provider.drawerDesc}
        />
        <Box>
          {provider.fields?.map(field => (
            <TextField
              variant="filled"
              autoComplete="off"
              key={field.name}
              name={field.name}
              label={field.label}
              defaultValue={field.value}
              helperText={field.helperText}
              fullWidth
              sx={{ mb: 2 }}
            />
          ))}

          <Switch
            checked={isEnabled}
            setChecked={setIsEnabled}
            name="status"
            label="Enable provider"
            description="Allow or disallow sign-in with this provider. Disabling will not delete existing users."
          />
        </Box>

        {provider.hasRedirectUrl && (
          <Box
            sx={{
              mt: 2,
              border: "1px solid",
              p: 2,
              borderRadius: 1,
              bgcolor: "container.paper",
              borderColor: "container.border",
            }}
          >
            <CopyBox
              fullWidth
              label="Callback URL"
              variant="filled"
              value={provider.redirectUrl}
              helperText="Set this URL as the Redirect URL in your OAuth provider settings."
            />
          </Box>
        )}

        <Box
          component="ul"
          sx={{
            my: 2,
            border: "1px solid",
            p: 2,
            borderRadius: 1,
            borderColor: "container.border",
          }}
        >
          {provider.steps?.map((step, index) => (
            <Typography
              component="li"
              key={index}
              sx={{ mb: 1, "&:last-child": { mb: 0 } }}
            >
              {index + 1}. {step}
            </Typography>
          ))}
        </Box>
        <CardFooter>
          <Button onClick={onClose} sx={{ mr: 2 }} variant="outlined">
            Cancel
          </Button>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={loading}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </Drawer>
  );
}
