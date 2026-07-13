import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import CardRow from "~/components/CardRow";
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";
import Api from "~/utils/api/Api";

interface Job {
  text: string;
  endpoint: string;
}

const jobs: Job[] = [
  {
    text: "Sync analytics last 30 days",
    endpoint: "/admin/jobs/sync-analytics?ts=30d",
  },
  {
    text: "Sync analytics last 7 days",
    endpoint: "/admin/jobs/sync-analytics?ts=7d",
  },
  {
    text: "Sync analytics last 24 hours",
    endpoint: "/admin/jobs/sync-analytics?ts=24h",
  },
  {
    text: "Remove old artifacts",
    endpoint: "/admin/jobs/remove-old-artifacts",
  },
];

export default function Jobs() {
  const [isLoading, setLoading] = useState<number>(-1);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();

  return (
    <Box>
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert severity="success" sx={{ mb: 2 }}>
          {success}
        </Alert>
      )}
      {jobs.map((job, index) => (
        <CardRow
          key={index}
          actions={
            <Button
              variant="contained"
              color="secondary"
              size="small"
              loading={isLoading === index}
              onClick={() => {
                setLoading(index);

                Api.post<{ deleted?: string[] }>(job.endpoint)
                  .then((res) => {
                    const count = res?.deleted?.length;
                    setSuccess(
                      count
                        ? `Deleted ${count} record${count === 1 ? "" : "s"} successfully.`
                        : "Job has been successfully run."
                    );
                    setError(undefined);
                  })
                  .catch(() => {
                    setError("Something went wrong while running the job.");
                    setSuccess(undefined);
                  })
                  .finally(() => {
                    setLoading(-1);
                  });
              }}
            >
              Sync
            </Button>
          }
        >
          <Typography>{job.text}</Typography>
        </CardRow>
      ))}
    </Box>
  );
}
