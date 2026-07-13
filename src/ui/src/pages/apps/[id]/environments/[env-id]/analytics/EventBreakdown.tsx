import type { TimeSpan } from "./index.d";
import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Alert from "@mui/material/Alert";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import LinearProgress from "@mui/material/LinearProgress";
import ArrowBack from "@mui/icons-material/ArrowBackIos";
import CardRow from "~/components/CardRow";
import { useFetchEventBreakdown, useFetchEventProperties } from "./actions";
import { truncate } from "./helpers";

interface Props {
  envId: string;
  domainId?: string;
  event: string;
  ts: TimeSpan;
  onBack: () => void;
}

export default function EventBreakdown({
  envId,
  domainId,
  event,
  ts,
  onBack,
}: Props) {
  const [property, setProperty] = useState("");

  const { properties, loading: propsLoading } = useFetchEventProperties({
    envId,
    domainId,
    event,
    ts,
  });

  const { breakdown, loading } = useFetchEventBreakdown({
    envId,
    domainId,
    event,
    property,
    ts,
    skip: property === "",
  });

  useEffect(() => {
    if (!property && properties.length) {
      setProperty(properties[0]);
    }
  }, [properties, property]);

  const hasProperties = propsLoading || properties.length > 0;

  return (
    <Box>
      <CardRow
        actions={
          <IconButton size="small" aria-label="Back to events" onClick={onBack}>
            <ArrowBack sx={{ fontSize: 14 }} />
          </IconButton>
        }
      >
        <Typography component="span">{truncate(event)}</Typography>
      </CardRow>

      {!hasProperties ? (
        <Alert color="info" sx={{ m: 2 }}>
          This event has no properties to group by yet. Attach a property with
          the second argument of track(), e.g. {`{ ref: "mobile" }`}.
        </Alert>
      ) : (
        <>
          <Box
            sx={{ px: 2, py: 1, display: "flex", alignItems: "center", gap: 1 }}
          >
            <Typography sx={{ fontSize: 14, opacity: 0.6 }}>Group by</Typography>
            <Select
              size="small"
              value={property}
              onChange={e => setProperty(e.target.value)}
              aria-label="Property"
            >
              {properties.map(p => (
                <MenuItem key={p} value={p}>
                  {p}
                </MenuItem>
              ))}
            </Select>
          </Box>

          {loading && <LinearProgress color="secondary" />}

          <Box sx={{ maxHeight: "260px", overflow: "auto" }}>
            {breakdown.map(row => (
              <CardRow
                key={row.name}
                chipLabel={
                  <Typography component="span">
                    {row.total.toLocaleString()}
                  </Typography>
                }
              >
                <Typography component="span">{truncate(row.name)}</Typography>
                <Typography
                  component="span"
                  sx={{ ml: 1, opacity: 0.6, fontSize: 12 }}
                >
                  · {row.unique.toLocaleString()} unique
                </Typography>
              </CardRow>
            ))}
          </Box>
        </>
      )}
    </Box>
  );
}
