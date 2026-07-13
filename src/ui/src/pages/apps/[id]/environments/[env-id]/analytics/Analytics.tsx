import type { TimeSpan } from "./index.d";
import { useContext, useState } from "react";
import { useSearchParams } from "react-router-dom";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Box from "@mui/material/Box";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import EmptyPage from "~/components/EmptyPage";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import DomainSelector from "~/shared/domains/DomainSelector";
import Visitors from "./Visitors";
import Events from "./Events";
import TopReferrers from "./TopReferrers";
import TopPaths from "./TopPaths";
import ByCountries from "./Countries";
import { RootContext } from "~/pages/Root.context";

export default function Analytics() {
  const { details } = useContext(RootContext);
  const [timeSpan, setTimeSpan] = useState<TimeSpan>("24h");
  const [params, setParams] = useSearchParams();
  const [domain, setDomain] = useState<Domain>();
  const [hasDomains, setHasDomains] = useState<boolean>();
  const { environment } = useContext(EnvironmentContext);
  const isCE = details?.license?.edition === "community";

  if (hasDomains === false || isCE) {
    return (
      <Card sx={{ width: "100%" }}>
        <CardHeader title="Analytics" />
        <EmptyPage paymentRequired={isCE}>
          Analytics are collected only for custom domains. <br />
          <Button
            sx={{ mt: 2 }}
            variant="contained"
            color="secondary"
            onClick={e => {
              e.preventDefault();
            }}
          >
            <Link
              href={`/apps/${environment.appId}/environments/${environment.id}#domains`}
              sx={{ ":hover": { color: "inherit", textDecoration: "none" } }}
            >
              Setup a custom domain
            </Link>
          </Button>
        </EmptyPage>
      </Card>
    );
  }

  const selectedDomain = domain?.domainName || params.get("domain");

  return (
    <Box>
      <Card sx={{ mb: 2 }}>
        <CardHeader
          title="Analytics"
          subtitle="Monitor user analytics for the specified domain within this environment configuration."
          actions={
            <DomainSelector
              selected={selectedDomain ? [selectedDomain] : []}
              appId={environment.appId}
              envId={environment.id!}
              fullWidth={false}
              onFetch={domains => {
                if (domains?.[0]) {
                  setDomain(
                    domains.find(d => d.domainName === params.get("domain")) ||
                      domains[0],
                  );

                  setHasDomains(true);
                } else if (typeof hasDomains === "undefined") {
                  setHasDomains(false);
                }
              }}
              onDomainSelect={d => {
                const selectedDomain = d ? (d[0] as Domain) : undefined;
                setDomain(selectedDomain);
                setParams({ domain: selectedDomain?.domainName || "" });
              }}
            />
          }
        />
      </Card>
      <Visitors
        domain={domain!}
        environment={environment}
        ts={timeSpan}
        onTimeSpanChange={setTimeSpan}
      />
      <Events domain={domain!} environment={environment} ts={timeSpan} />
      <Box
        sx={{
          mt: 2,
          display: "flex",
          flexDirection: { xs: "column", md: "row" },
          gap: 2,
        }}
      >
        <TopReferrers environment={environment} domain={domain!} />
        <TopPaths environment={environment} domain={domain!} />
      </Box>
      <ByCountries environment={environment} domain={domain!} />
    </Box>
  );
}
