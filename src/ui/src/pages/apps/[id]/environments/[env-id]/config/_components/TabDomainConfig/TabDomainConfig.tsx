import { useMemo, useState } from "react";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import ArrowRightIcon from "@mui/icons-material/ArrowRight";
import HttpsIcon from "@mui/icons-material/Https";
import WarningIcon from "@mui/icons-material/Warning";
import { yellow } from "@mui/material/colors";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import EmptyPage from "~/components/EmptyPage";
import ConfirmModal from "~/components/ConfirmModal";
import Span from "~/components/Span";
import IconBg from "~/components/IconBg";
import Dot from "~/components/Dot";
import { isSelfHosted } from "~/utils/helpers/instance";
import DomainModal from "./DomainModal";
import DomainVerifyModal from "./DomainVerifyModal";
import CustomCertModal from "./CustomCertModal";
import { deleteDomain, useFetchDomains } from "./actions";
import { formattedDate } from "~/utils/helpers/deployments";

interface Props {
  app: App;
  environment: Environment;
}

export default function TabDomainConfig({ app, environment }: Props) {
  const selfHosted = useMemo(() => {
    return isSelfHosted();
  }, []);

  const [afterId, setAfterId] = useState<string>();
  const [refreshToken, setRefreshToken] = useState(0);
  const [isDomainModalOpen, toggleDomainModal] = useState(false);
  const [domainToModifyCustomCert, toggleCustomCertModal] = useState<Domain>();
  const [success, setSuccess] = useState<string>();
  const [domainToVerify, setDomainToVerify] = useState<Domain>();
  const [domainToDelete, setDomainToDelete] = useState<Domain>();
  const { domains, error, loading, pagination } = useFetchDomains({
    appId: app.id,
    envId: environment.id!,
    afterId,
    refreshToken,
    search: "",
  });

  return (
    <Card error={error} success={success} loading={loading}>
      <CardHeader
        title="Custom domains"
        subtitle="Set custom domains to serve your application from."
      />
      {!selfHosted && (
        <Alert color="info" sx={{ px: 4 }}>
          Use the following A Record to point your domains:
          <Box sx={{ mt: 1 }}>
            <ArrowRightIcon /> 91.98.218.209
          </Box>
        </Alert>
      )}
      <Box>
        {domains.map(domain => (
          <CardRow
            key={domain.id}
            menuItems={[
              {
                text: "Verify now",
                hidden: domain.verified,
                onClick: () => {
                  setDomainToVerify(domain);
                },
              },
              {
                text: "Custom certificate",
                onClick: () => toggleCustomCertModal(domain),
              },
              {
                text: "Delete",
                onClick: () => setDomainToDelete(domain),
              },
            ]}
          >
            <Box sx={{ flex: 1, display: "flex", alignItems: "center" }}>
              <Typography
                sx={{
                  display: "flex",
                  alignItems: "center",
                  width: "100%",
                  justifyContent: "space-between",
                }}
              >
                <Box
                  component="span"
                  sx={{ display: "inline-flex", alignItems: "center" }}
                >
                  <IconBg>
                    {domain.verified ? (
                      <HttpsIcon
                        color={
                          domain.lastPing?.status?.toString()[0] == "2"
                            ? "success"
                            : domain.lastPing?.status
                              ? "error"
                              : "warning"
                        }
                        sx={{ fontSize: 14 }}
                      />
                    ) : (
                      <WarningIcon sx={{ color: yellow[500], fontSize: 14 }} />
                    )}
                  </IconBg>
                  {domain.domainName}
                  {domain.customCert && (
                    <Typography component="span" color="text.secondary">
                      <Dot sx={{ mx: 1 }} />
                      Custom certificate
                    </Typography>
                  )}
                </Box>
                <Tooltip
                  title={`${
                    domain.lastPing
                      ? `Last ping ${formattedDate(
                          domain.lastPing.lastPingAt!,
                        ).toLowerCase()}`
                      : domain.verified
                        ? "Domain health is checked every 5 minutes. Refresh to see the latest status."
                        : ""
                  }`}
                >
                  <Box
                    component="span"
                    data-testid={`${domain.domainName}-status`}
                  >
                    <Span sx={{ m: 0 }}>
                      Status:{" "}
                      {domain.lastPing?.status
                        ? domain.lastPing.status
                        : domain.verified
                          ? "not yet pinged"
                          : "not yet verified"}
                    </Span>
                  </Box>
                </Tooltip>
              </Typography>
            </Box>
          </CardRow>
        ))}
      </Box>
      {!loading && !error && !domains.length && (
        <EmptyPage>
          <Typography component="span" sx={{ display: "block" }}>
            No custom domain configuration found.
          </Typography>
          <Typography component="span">
            Add a domain to serve your app directly from it.
          </Typography>
        </EmptyPage>
      )}
      <CardFooter>
        {pagination?.hasNextPage && (
          <Button
            type="button"
            variant="text"
            color="info"
            sx={{ mr: 2 }}
            onClick={() => {
              setAfterId(pagination?.afterId);
            }}
          >
            Load more
          </Button>
        )}
        <Button
          color="secondary"
          variant="contained"
          onClick={() => toggleDomainModal(true)}
          type="button"
        >
          Add domain
        </Button>
      </CardFooter>
      {domainToDelete && (
        <ConfirmModal
          onCancel={() => setDomainToDelete(undefined)}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            deleteDomain({
              appId: app.id,
              envId: environment.id!,
              domainId: domainToDelete.id,
            })
              .then(() => {
                setDomainToDelete(undefined);
                setRefreshToken(Date.now());
              })
              .catch(res => {
                setError(
                  res.status === 400
                    ? "Please provide a valid domain name."
                    : res.status === 429
                      ? "You have issued too many requests. Please wait a while before retrying."
                      : "Something went wrong while setting up the domain. Make sure it is a valid domain.",
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          This will completely remove the domain and it won't be reachable
          anymore.
        </ConfirmModal>
      )}
      {isDomainModalOpen && (
        <DomainModal
          setRefreshToken={setRefreshToken}
          onClose={() => toggleDomainModal(false)}
        />
      )}
      {domainToModifyCustomCert && (
        <CustomCertModal
          appId={app.id}
          envId={environment.id!}
          domain={domainToModifyCustomCert}
          setSuccess={setSuccess}
          onClose={() => toggleCustomCertModal(undefined)}
          onUpdate={() => {
            setRefreshToken(Date.now());
            toggleCustomCertModal(undefined);
          }}
        />
      )}
      {domainToVerify && (
        <DomainVerifyModal
          domain={domainToVerify}
          environment={environment}
          app={app}
          onClose={() => {
            setDomainToVerify(undefined);
          }}
        />
      )}
    </Card>
  );
}
