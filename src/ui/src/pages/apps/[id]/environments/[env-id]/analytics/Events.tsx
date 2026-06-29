import type { TimeSpan } from "./index.d";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import { useFetchEvents } from "./actions";
import { truncate } from "./helpers";

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

export default function Events({ environment, domain, ts }: Props) {
  const { events, error, loading } = useFetchEvents({
    envId: environment.id!,
    domainId: domain?.id,
    ts,
  });

  const isEmpty = !loading && !error && events.length === 0;

  return (
    <Card
      sx={{ mt: 2 }}
      error={error}
      loading={loading}
      info={
        isEmpty ? (
          <>
            No events yet. Track events from the browser with{" "}
            <Box component="code">window.stormkit.track("event_name")</Box> or
            by posting to <Box component="code">/_stormkit/collect</Box>.
          </>
        ) : undefined
      }
    >
      <CardHeader
        title="Events"
        subtitle="Custom events tracked for this domain."
      />
      <Box sx={{ maxHeight: "300px", overflow: "auto" }}>
        {events.map(event => (
          <CardRow
            key={event.name}
            chipLabel={
              <Typography component="span">
                {event.total.toLocaleString()}
              </Typography>
            }
          >
            <Typography component="span">{truncate(event.name)}</Typography>
            <Typography
              component="span"
              sx={{ ml: 1, opacity: 0.6, fontSize: 12 }}
            >
              · {event.unique.toLocaleString()} unique
            </Typography>
          </CardRow>
        ))}
      </Box>
    </Card>
  );
}
