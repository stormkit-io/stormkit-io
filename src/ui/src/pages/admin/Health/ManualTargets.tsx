import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import { updateTargets, useFetchTargets } from "./actions";

interface Props {
  onUpdate: () => void;
}

/**
 * Machines running Stormkit register themselves, so this covers only the rest —
 * a database host, for instance, that runs an exporter but no Stormkit process.
 */
export default function ManualTargets({ onUpdate }: Props) {
  const [refreshToken, setRefreshToken] = useState(0);
  const { targets, loading } = useFetchTargets({ refreshToken });
  const [value, setValue] = useState("");
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue(targets.join("\n"));
  }, [targets]);

  return (
    <Card
      loading={loading}
      error={error}
      success={success}
      component="form"
      sx={{ mb: 2 }}
      onSubmit={e => {
        e.preventDefault();

        setError(undefined);
        setSuccess(undefined);
        setSaving(true);

        updateTargets(
          value
            .split("\n")
            .map(t => t.trim())
            .filter(Boolean),
        )
          .then(() => {
            setSuccess("Targets updated.");
            setRefreshToken(Date.now());
            onUpdate();
          })
          .catch(async res => {
            const body = await res?.json?.().catch(() => ({}));

            setError(
              body?.error ||
                "Something went wrong while updating targets. Please try again.",
            );
          })
          .finally(() => {
            setSaving(false);
          });
      }}
    >
      <CardHeader
        title="Manual targets"
        subtitle="Machines that run node_exporter but no Stormkit process. One host per line."
      />
      <CardRow>
        <TextField
          fullWidth
          multiline
          minRows={2}
          value={value}
          onChange={e => setValue(e.target.value)}
          placeholder="db-host:9100"
          slotProps={{ htmlInput: { "aria-label": "Manual targets" } }}
        />
        <Typography sx={{ fontSize: 12, mt: 1, opacity: 0.6 }}>
          Machines running Stormkit are discovered automatically and do not need
          to be listed here.
        </Typography>
      </CardRow>
      <CardFooter>
        <Button
          type="submit"
          variant="contained"
          color="secondary"
          disabled={saving}
        >
          Save
        </Button>
      </CardFooter>
    </Card>
  );
}
