import { useState, useContext } from "react";
import Box from "@mui/material/Box";
import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { formatDate } from "~/utils/helpers/date";
import { useFetchEmails, type Email } from "./actions";

export default function SentEmails() {
  const { app } = useContext(AppContext);
  const { environment: env } = useContext(EnvironmentContext);
  const [selectedEmail, setSelectedEmail] = useState<Email>();
  const { loading, error, emails } = useFetchEmails({
    appId: app.id,
    envId: env.id!,
  });

  return (
    <>
      <Card id="sent-emails" width="100%" loading={loading} error={error}>
        <CardHeader
          title="Sent Emails"
          subtitle="Last 100 emails sent from this environment."
        />

        {emails.map(email => (
          <CardRow
            key={email.id}
            sx={{ cursor: "pointer" }}
            onClick={() => setSelectedEmail(email)}
          >
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <Box sx={{ flex: 1 }}>
                <Typography>{email.subject}</Typography>
                <Typography sx={{ fontSize: 12, color: "text.secondary" }}>
                  To: {email.to}
                </Typography>
              </Box>
              <Typography
                sx={{ fontSize: 12, color: "text.secondary", ml: 2 }}
              >
                {formatDate(email.sentAt)}
              </Typography>
            </Box>
          </CardRow>
        ))}

        {!loading && emails.length === 0 && (
          <Box sx={{ px: 4, py: 2 }}>
            <Typography sx={{ color: "text.secondary" }}>
              No emails sent yet.
            </Typography>
          </Box>
        )}
      </Card>

      {selectedEmail && (
        <Dialog
          open
          onClose={() => setSelectedEmail(undefined)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>{selectedEmail.subject}</DialogTitle>
          <DialogContent>
            <Box sx={{ mb: 2 }}>
              <Typography variant="body2" color="text.secondary">
                From: {selectedEmail.from}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                To: {selectedEmail.to}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Sent: {formatDate(selectedEmail.sentAt)}
              </Typography>
            </Box>
            <Box
              component="iframe"
              srcDoc={selectedEmail.body}
              sx={{ width: "100%", minHeight: 400, border: "none" }}
              title="Email preview"
            />
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}
