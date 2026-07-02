import type { TimeSpan } from "./index.d";
import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import ArrowForward from "@mui/icons-material/ArrowForwardIos";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import Help from "~/components/Help";
import { useFetchEvents } from "./actions";
import { truncate } from "./helpers";
import EventsHelpContent from "./EventsHelpContent";
import EventBreakdown from "./EventBreakdown";
import AnalyticsToggle from "./AnalyticsToggle";

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

const helpTitle = "Track events";
const helpSubtitle = "Send custom events from the browser or from your backend.";

export default function Events({ environment, domain, ts }: Props) {
  const [selected, setSelected] = useState("");
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
            No events yet.{" "}
            <Help
              title={helpTitle}
              subtitle={helpSubtitle}
              buttonText="Learn how to send events."
              buttonVariant="link"
            >
              <EventsHelpContent domain={domain} />
            </Help>
          </>
        ) : undefined
      }
    >
      <CardHeader
        title="Events"
        subtitle="Custom events tracked for this domain."
        actions={
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <AnalyticsToggle
              appId={environment.appId}
              envId={environment.id!}
            />
            <Help
              title={helpTitle}
              subtitle={helpSubtitle}
              buttonText="How to track"
              buttonVariant="text"
              iconOnly={!isEmpty}
            >
              <EventsHelpContent domain={domain} />
            </Help>
          </Box>
        }
      />
      {selected ? (
        <EventBreakdown
          envId={environment.id!}
          domainId={domain?.id}
          event={selected}
          ts={ts}
          onBack={() => setSelected("")}
        />
      ) : (
        <Box sx={{ maxHeight: "300px", overflow: "auto" }}>
          {events.map(event => (
            <CardRow
              key={event.name}
              chipLabel={
                <Typography component="span">
                  {event.total.toLocaleString()}
                </Typography>
              }
              actions={
                <IconButton
                  size="small"
                  sx={{ ml: 2 }}
                  aria-label={`Group ${event.name}`}
                  onClick={() => setSelected(event.name)}
                >
                  <ArrowForward sx={{ fontSize: 12 }} />
                </IconButton>
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
      )}
    </Card>
  );
}
