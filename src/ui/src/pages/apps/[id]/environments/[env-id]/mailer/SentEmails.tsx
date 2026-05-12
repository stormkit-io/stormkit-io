import { useState, useContext } from "react";
import Box from "@mui/material/Box";
import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
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
      <Card
        id="sent-emails"
        width="100%"
        loading={loading}
        error={error}
        sx={{ pb: 4 }}
        contentPadding={false}
        info={emails.length === 0 ? "No emails sent yet." : undefined}
      >
        <CardHeader
          title="Sent Emails"
          subtitle="Last 100 emails sent from this environment."
        />

        {emails.map(email => (
          <CardRow key={email.id}>
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <Box sx={{ flex: 1 }}>
                <Link href="#" onClick={() => setSelectedEmail(email)}>
                  {email.subject}
                </Link>
                <Typography sx={{ fontSize: 12, color: "text.secondary" }}>
                  To: {email.to}
                </Typography>
              </Box>
              <Typography sx={{ fontSize: 12, color: "text.secondary", ml: 2 }}>
                {formatDate(email.sentAt)}
              </Typography>
            </Box>
          </CardRow>
        ))}
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
                From: {selectedEmail.from || "-"}
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
              bgcolor="container.paper"
              border="1px solid"
              borderColor="container.border"
              srcDoc={`<style>*,a{color:#fff!important}</style>${selectedEmail.body}`}
              sx={{
                p: 2,
                borderRadius: 1,
                width: "100%",
                minHeight: 400,
              }}
              title="Email preview"
            />
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}
