import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import api from "~/utils/api/Api";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useFetchInstanceMailer } from "./actions";

export default function Mailer() {
  const { user } = useContext(AuthContext);
  const [refreshToken, setRefreshToken] = useState<number>();
  const [formError, setFormError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isSaving, setIsSaving] = useState(false);
  const [isSendingTest, setIsSendingTest] = useState(false);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const { mailer, loading, error } = useFetchInstanceMailer({ refreshToken });

  const reportError = async (res: Response) => {
    setSuccess(undefined);
    setFormError((await api.errors(res)).join(" "));
  };

  const handleSave: React.FormEventHandler = e => {
    e.preventDefault();

    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries());

    setFormError(undefined);
    setSuccess(undefined);
    setIsSaving(true);

    api
      .post("/admin/mailer", data)
      .then(() => {
        setSuccess("The default mailer is saved.");
        setRefreshToken(Date.now());
      })
      .catch(reportError)
      .finally(() => {
        setIsSaving(false);
      });
  };

  const handleSendTest = () => {
    setFormError(undefined);
    setSuccess(undefined);
    setIsSendingTest(true);

    api
      .post("/admin/mailer/test", { to: user?.email })
      .then(() => {
        setSuccess(`A test email is sent to ${user?.email}.`);
      })
      .catch(reportError)
      .finally(() => {
        setIsSendingTest(false);
      });
  };

  return (
    <Card
      component="form"
      loading={loading}
      error={error || formError}
      success={success}
      onSubmit={handleSave}
      sx={{ backgroundColor: "container.transparent" }}
      contentPadding={false}
    >
      <CardHeader
        title="Default mailer"
        subtitle="The SMTP account environments can use instead of their own. Emails sent through it always come from the address below, and its credentials are never given to app code."
      />
      {/* Remount the fields once the stored mailer is loaded or saved. */}
      <Box key={JSON.stringify(mailer)} sx={{ px: 4 }}>
        <TextField
          label="SMTP Host"
          name="host"
          defaultValue={mailer?.host || ""}
          variant="filled"
          autoComplete="off"
          fullWidth
          sx={{ mb: 4 }}
        />
        <TextField
          label="SMTP Port"
          name="port"
          defaultValue={mailer?.port || ""}
          placeholder="587"
          variant="filled"
          autoComplete="off"
          fullWidth
          sx={{ mb: 4 }}
        />
        <TextField
          label="Username"
          name="username"
          defaultValue={mailer?.username || ""}
          variant="filled"
          autoComplete="off"
          fullWidth
          sx={{ mb: 4 }}
        />
        <TextField
          type="password"
          label="Password"
          name="password"
          defaultValue=""
          helperText={
            mailer?.password
              ? "A password is stored. Leave this empty to keep it — changing the host or username requires re-entering it."
              : undefined
          }
          variant="filled"
          autoComplete="new-password"
          fullWidth
          sx={{ mb: 4 }}
        />
        <TextField
          label="From"
          name="from"
          defaultValue={mailer?.from || ""}
          placeholder="Acme <noreply@acme.com>"
          helperText="The sender of every email sent through the default mailer. Your SMTP provider must allow sending from it."
          variant="filled"
          autoComplete="off"
          fullWidth
          sx={{ mb: 4 }}
        />
      </Box>
      <CardFooter>
        {mailer && (
          <>
            <Button
              type="button"
              variant="text"
              color="info"
              sx={{ mr: 2 }}
              onClick={() => setIsConfirmOpen(true)}
            >
              Remove
            </Button>
            <Button
              type="button"
              variant="text"
              color="info"
              sx={{ mr: 2 }}
              loading={isSendingTest}
              onClick={handleSendTest}
            >
              Send test email
            </Button>
          </>
        )}
        <Button
          type="submit"
          variant="contained"
          color="secondary"
          loading={isSaving}
        >
          Save
        </Button>
      </CardFooter>
      {isConfirmOpen && (
        <ConfirmModal
          onCancel={() => setIsConfirmOpen(false)}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);

            api
              .delete("/admin/mailer")
              .then(() => {
                setIsConfirmOpen(false);
                setSuccess("The default mailer is removed.");
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setLoading(false);
                setError("Something went wrong while removing the mailer.");
              });
          }}
        >
          Environments that use the default mailer will record their emails
          without sending them until a default mailer is configured again.
        </ConfirmModal>
      )}
    </Card>
  );
}
