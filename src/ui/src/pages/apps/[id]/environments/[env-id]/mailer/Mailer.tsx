import { useState, useContext } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";
import Help, { HelpTable } from "~/components/Help";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { useFetchMailerConfig } from "./actions";

export default function TabMailer() {
  const { app } = useContext(AppContext);
  const { environment: env } = useContext(EnvironmentContext);
  const [refreshToken, setRefreshToken] = useState<number>();
  const [formError, setFormError] = useState<string>();
  const [sendLoading, setSendLoading] = useState<boolean>(false);
  const [success, setSuccess] = useState<string>();
  const { loading, error, config } = useFetchMailerConfig({
    appId: app.id,
    envId: env.id!,
    refreshToken,
  });

  const handleUpdateConfig: React.FormEventHandler = e => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    if (data["username"] === "") {
      return setFormError("Username is a required field");
    }

    api
      .post("/mailer/config", { appId: app.id, envId: env.id!, ...data })
      .then(() => {
        setSuccess("Mailer configuration saved successfully.");
        setRefreshToken(Date.now());
      });
  };

  return (
    <Card
      id="mailer"
      component="form"
      width="100%"
      loading={loading}
      error={error || formError}
      success={success}
      onSubmit={handleUpdateConfig}
      sx={{ "div[data-lastpass-icon-root]": { display: "none" } }}
    >
      <CardHeader
        title="Mailer Configuration"
        subtitle={
          <>
            Simple Email Service to send transactional emails.{" "}
            <Help
              title="Mailer Configuration Help"
              buttonText="Learn more."
              buttonVariant="link"
            >
              <Box>
                <Typography>
                  Configure your SMTP settings to enable transactional emails
                  from your application. Common SMTP providers include Gmail,
                  SendGrid, Mailgun, and Amazon SES.
                </Typography>
                <Box sx={{ my: 4 }}>
                  <Typography variant="h2" sx={{ mb: 1 }}>
                    Environment Variables
                  </Typography>
                  <Typography sx={{ mb: 2 }}>
                    When configured, Stormkit will inject the following
                    environment variable at build time and make it available at
                    runtime:
                  </Typography>
                  <HelpTable
                    items={[
                      {
                        name: <Box component="code">MAILER_URL</Box>,
                        desc: "The API endpoint for sending emails",
                      },
                    ]}
                  />
                  <Typography sx={{ mt: 2 }}>
                    Note that if an environment already has a custom{" "}
                    <Box component="code">MAILER_URL</Box> configured, it won't
                    be overwritten.
                  </Typography>
                </Box>
                <Box sx={{ my: 4 }}>
                  <Typography variant="h2" sx={{ mb: 1 }}>
                    Example Configuration (Gmail)
                  </Typography>
                  <HelpTable
                    items={[
                      {
                        name: "SMTP Host",
                        desc: <Box component="code">smtp.gmail.com</Box>,
                      },
                      {
                        name: "SMTP Port",
                        desc: <Box component="code">587</Box>,
                      },
                      {
                        name: "Username",
                        desc: <Box component="code">your-email@gmail.com</Box>,
                      },
                      {
                        name: "Password",
                        desc: "Your Gmail app password (required if 2FA is enabled)",
                      },
                    ]}
                  />
                </Box>
                <Typography>
                  For programmatic email sending, generate an API key under{" "}
                  <strong>Environment &gt; Config &gt; API Keys</strong> and
                  refer to our{" "}
                  <Link
                    href="https://www.stormkit.io/docs/api/mailer"
                    color="text.secondary"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    API documentation
                  </Link>
                  .
                </Typography>
              </Box>
            </Help>
          </>
        }
      />

      <Box sx={{ mb: 4 }}>
        <TextField
          label="SMTP Host"
          name="smtpHost"
          fullWidth
          defaultValue={config?.host || ""}
          variant="filled"
          autoComplete="off"
        />
      </Box>

      <Box sx={{ mb: 4 }}>
        <TextField
          label="SMTP Port"
          name="smtpPort"
          fullWidth
          defaultValue={config?.port || ""}
          variant="filled"
          autoComplete="off"
        />
      </Box>

      <Box sx={{ mb: 4 }}>
        <TextField
          label="Username"
          name="username"
          fullWidth
          defaultValue={config?.username || ""}
          variant="filled"
          autoComplete="off"
        />
      </Box>

      <Box sx={{ mb: 4 }}>
        <TextField
          type="password"
          label="Password"
          name="password"
          fullWidth
          defaultValue={config?.password || ""}
          variant="filled"
          autoComplete="off"
        />
      </Box>

      <CardFooter>
        {config && (
          <Button
            type="button"
            variant="text"
            color="info"
            loading={sendLoading}
            sx={{ mr: 2 }}
            onClick={() => {
              setSendLoading(true);

              api
                .post("/mailer", {
                  // fetch treats the `body` argument as a json string so we
                  // need to stringify the parameters to make this api call work.
                  body: JSON.stringify({
                    appId: app.id,
                    envId: env.id!,
                    from: config.username,
                    to: config.username,
                    body: "Test email body",
                    subject: "Test email subject",
                  }),
                })
                .then(() => {
                  setSuccess("Test email sent to " + config.username);
                })
                .catch(() => {
                  setFormError(
                    "Something went wrong while sending test email.",
                  );
                })
                .finally(() => {
                  setSendLoading(false);
                });
            }}
          >
            Send test email
          </Button>
        )}

        <Button type="submit" variant="contained" color="secondary">
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}
