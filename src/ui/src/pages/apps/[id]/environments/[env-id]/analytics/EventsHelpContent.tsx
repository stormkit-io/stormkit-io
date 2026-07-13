import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import CopyBox from "~/components/CopyBox";

interface Props {
  domain?: Domain;
}

const clientExample = `window.stormkit.track("new_trip_creation", { ref: "mobile" });`;

function serverExample(host: string) {
  return [
    `curl -X POST https://${host}/_stormkit/collect \\`,
    `  -H 'Content-Type: application/json' \\`,
    `  -d '{"events":[{"name":"new_trip_creation","metadata":{"ref":"mobile"}}]}'`,
  ].join("\n");
}

export default function EventsHelpContent({ domain }: Props) {
  const host = domain?.domainName || "<your-domain>";

  return (
    <Box sx={{ px: 4, pb: 4 }}>
      <Typography sx={{ mb: 1, fontWeight: 500 }}>From the browser</Typography>
      <Typography sx={{ mb: 2, opacity: 0.6, fontSize: 14 }}>
        Requires the Stormkit analytics script to be enabled for this
        environment.
      </Typography>
      <CopyBox value={clientExample} multiline />

      <Typography sx={{ mt: 4, mb: 1, fontWeight: 500 }}>
        From your server
      </Typography>
      <Typography sx={{ mb: 2, opacity: 0.6, fontSize: 14 }}>
        Post directly to the collect endpoint — useful for backend events that
        never reach the browser.
      </Typography>
      <CopyBox value={serverExample(host)} multiline minRows={3} />

      <Typography sx={{ mt: 4, opacity: 0.6, fontSize: 14 }}>
        The second argument is an arbitrary key-value object, stored as{" "}
        <code>metadata</code> with each event. Every property can be queried by
        its name — group an event by any property, e.g. break{" "}
        <code>new_trip_creation</code> down by <code>ref</code> or{" "}
        <code>platform</code>.
      </Typography>
    </Box>
  );
}
