import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import PersonalAccessTokenModal from "./PersonalAccessTokenModal";
import { useFetchEmails } from "../actions";

interface Props {
  accounts: ConnectedAccount[];
}

const texts: Record<Provider, string> = {
  github: "GitHub",
  gitlab: "GitLab",
  bitbucket: "Bitbucket",
};

const showPersonalAccessButton = (provider: Provider): boolean => {
  return provider === "gitlab";
};

export default function ConnectedAccounts({ accounts }: Props) {
  const [isOpen, toggleModal] = useState(false);
  const { emails, loading, error } = useFetchEmails();

  return (
    <Card error={error} loading={loading} sx={{ my: 2 }} contentPadding={false}>
      <CardHeader
        title="Connected Accounts"
        subtitle={
          <>
            List of connected emails and providers
            <br /> We combine providers using shared email addresses
          </>
        }
      />
      <Box>
        <Typography sx={{ mb: 2, px: 4 }}>Providers</Typography>
        {accounts.map(({ provider, hasPersonalAccessToken }) => (
          <CardRow
            key={provider}
            data-testid={provider}
            actions={
              showPersonalAccessButton(provider) ? (
                <>
                  <Button
                    variant="text"
                    color="info"
                    size="small"
                    onClick={e => {
                      e.preventDefault();
                      toggleModal(true);
                    }}
                  >
                    {hasPersonalAccessToken ? "Reset" : "Set"} personal access
                    token
                  </Button>
                  {isOpen && (
                    <PersonalAccessTokenModal
                      hasToken={hasPersonalAccessToken}
                      toggleModal={toggleModal}
                    />
                  )}
                </>
              ) : undefined
            }
          >
            <Typography sx={{ color: "text.secondary" }}>
              {texts[provider]}
            </Typography>
          </CardRow>
        ))}
        <Typography sx={{ mt: 2, px: 4 }}>Emails</Typography>
        {emails?.map(email => (
          <CardRow
            key={email.address}
            data-testid={email.address}
            chipLabel={email.primary ? "Primary" : undefined}
            chipColor={email.primary ? "success" : undefined}
          >
            <Typography sx={{ color: "text.secondary" }}>
              {email.address}
            </Typography>
          </CardRow>
        ))}
      </Box>
    </Card>
  );
}
