import { useState } from "react";
import { parse as parseUrl } from "tldts";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import CardFooter from "~/components/CardFooter";
import CardHeader from "~/components/CardHeader";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardRow from "~/components/CardRow";
import CopyBox from "~/components/CopyBox";
import { useDomainLookup } from "./actions";

interface Props {
  domain: Domain;
  app: App;
  environment: Environment;
  onClose: () => void;
}

export default function DomainVerifyModal({
  app,
  environment,
  domain,
  onClose,
}: Props) {
  const [refreshToken, setRefreshToken] = useState(0);
  const { error, loading, info } = useDomainLookup({
    appId: app.id,
    envId: environment.id!,
    domainId: domain.id,
    refreshToken,
  });

  return (
    <Modal open onClose={onClose}>
      <Card
        loading={loading}
        error={error}
        contentPadding={false}
        info={
          refreshToken > 0 && !info?.dns.verified
            ? "Domain is still not yet verified. If you updated your TXT record, please give it a bit time to propagate."
            : ""
        }
      >
        <CardHeader
          title={domain.domainName}
          subtitle="Follow these steps to verify your domain"
        />
        {info && (
          <Box>
            <CardRow>
              <Typography sx={{ color: "text.secondary", mb: 2 }}>
                Status
              </Typography>
              <Chip
                color={info.dns.verified ? "success" : undefined}
                label={info.dns.verified ? "verified" : "not yet verified"}
                size="small"
              />
            </CardRow>

            <CardRow sx={{ mb: refreshToken > 0 ? 1 : 0 }}>
              <Typography sx={{ color: "text.secondary", mb: 2 }}>
                Login to your external DNS provider and create the following TXT
                record.
              </Typography>
              <CopyBox
                value={parseUrl(info.dns.txt.lookup).subdomain}
                slotProps={{
                  input: {
                    readOnly: true,
                  },
                }}
                fullWidth
                label="Txt Host"
                variant="outlined"
                margin="normal"
              />
              <CopyBox
                value={info.dns.txt.value}
                InputProps={{
                  readOnly: true,
                }}
                fullWidth
                label="Txt Value"
                variant="outlined"
                margin="normal"
              />
            </CardRow>
          </Box>
        )}
        <CardFooter>
          <Button
            color="secondary"
            variant="contained"
            loading={loading}
            onClick={() => {
              setRefreshToken(Date.now());
            }}
          >
            Verify now
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
