import { useState } from "react";
import Switch from "@mui/material/Switch";
import FormControlLabel from "@mui/material/FormControlLabel";
import ConfirmModal from "~/components/ConfirmModal";
import {
  useFetchAnalyticsStatus,
  enableAnalytics,
  disableAnalytics,
} from "./actions";

interface Props {
  appId: string;
  envId: string;
}

export default function AnalyticsToggle({ appId, envId }: Props) {
  const [refreshToken, setRefreshToken] = useState<number>();
  const [confirm, setConfirm] = useState(false);
  const { enabled, loading } = useFetchAnalyticsStatus({
    appId,
    envId,
    refreshToken,
  });

  return (
    <>
      <FormControlLabel
        sx={{ mr: 0 }}
        labelPlacement="start"
        label="Client-side tracking"
        control={
          <Switch
            color="secondary"
            checked={enabled}
            disabled={loading}
            onChange={() => setConfirm(true)}
            inputProps={{ "aria-label": "Toggle client-side tracking" }}
          />
        }
      />
      {confirm && (
        <ConfirmModal
          onCancel={() => setConfirm(false)}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);

            const toggle = enabled ? disableAnalytics : enableAnalytics;

            toggle({ appId, envId })
              .then(() => {
                setRefreshToken(Date.now());
                setConfirm(false);
              })
              .catch(() => {
                setError(
                  "Something went wrong while updating client-side tracking. Please try again."
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          This will {enabled ? "disable" : "enable"} the Stormkit analytics
          script for this environment. The change takes effect immediately, on
          the next page load.
        </ConfirmModal>
      )}
    </>
  );
}
