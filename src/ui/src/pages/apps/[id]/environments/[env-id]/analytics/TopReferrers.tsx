import type { TimeSpan } from "./index.d";
import Box from "@mui/material/Box";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import LinearProgress from "@mui/material/LinearProgress";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import { useFetchAnalyticsReport } from "./actions";
import { channelLabels, spanText, truncate } from "./helpers";

const formatter = new Intl.NumberFormat();

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

export default function TopReferrers({ environment, domain, ts }: Props) {
  const { report, error, loading } = useFetchAnalyticsReport({
    envId: environment.id!,
    domainId: domain?.id,
    query: { metric: "referrers", ts, limit: 50 },
  });

  const referrers = report?.rows || [];
  const totals = report?.totals;

  return (
    <Card
      error={error}
      loading={loading && !report}
      sx={{
        width: { xs: "100%", md: "50%" },
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <CardHeader
        title="Referrers"
        subtitle={
          <>
            Top referring sites {spanText(ts)}.
            {totals && (
              <>
                {" "}
                {formatter.format(totals.direct || 0)} direct visits,{" "}
                {formatter.format(totals.internal || 0)} within this site.
              </>
            )}
          </>
        }
      />
      {loading && report && <LinearProgress color="secondary" />}
      <Box
        sx={{
          maxHeight: "300px",
          overflow: "auto",
          flex: 1,
        }}
      >
        {!report ? null : referrers.length ? (
          referrers.map(ref => (
            <CardRow
              key={ref.referrer}
              chipLabel={
                <Typography component="span">
                  {formatter.format(ref.visits || 0)}
                </Typography>
              }
            >
              <Typography component="span">
                {truncate(ref.referrer!)}
              </Typography>
              {ref.channel && (
                <Typography
                  component="span"
                  sx={{ ml: 1, fontSize: 12, opacity: 0.6 }}
                >
                  {channelLabels[ref.channel] || ref.channel}
                </Typography>
              )}
            </CardRow>
          ))
        ) : (
          <Alert
            color="info"
            sx={{ mx: 2, mb: 2, bgcolor: "rgba(255,255,255,0.025)" }}
          >
            No visits from other sites {spanText(ts)}.
          </Alert>
        )}
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          Visits from this environment&apos;s own domains are counted as within
          this site.
        </Typography>
      </CardFooter>
    </Card>
  );
}
