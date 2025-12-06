import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import AddIcon from "@mui/icons-material/Add";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import EmptyPage from "~/components/EmptyPage";
import CardFooter from "~/components/CardFooter";
import { useFetchDatabase } from "./actions";

interface EmptyViewProps {
  onAttachClick: () => void;
}

function EmptyView({ onAttachClick }: EmptyViewProps) {
  return (
    <EmptyPage>
      <Typography
        component="span"
        variant="h6"
        sx={{ mb: 4, display: "block" }}
      >
        No database attached to this environment
      </Typography>
      <Box component="span" sx={{ display: "block" }}>
        <Button
          href="https://www.stormkit.io/docs/features/database"
          variant="outlined"
          color="primary"
          target="_blank"
          rel="noreferrer noopener"
          endIcon={<OpenInNewIcon />}
        >
          Learn more
        </Button>
        <Button
          variant="contained"
          color="secondary"
          sx={{ ml: 2 }}
          onClick={onAttachClick}
          startIcon={<AddIcon />}
        >
          Attach Database
        </Button>
      </Box>
    </EmptyPage>
  );
}

export default function Database() {
  const { environment } = useContext(EnvironmentContext);
  const [refreshToken, _] = useState<number>();
  const { database, loading, error } = useFetchDatabase({
    envId: environment.id!,
    refreshToken,
  });
  const [success, setSuccess] = useState<string>();
  const [isAttachModalOpen, setIsAttachModalOpen] = useState(false);

  // If the request is not loading and there is no error,
  // and the database exists, then a database is attached.
  const hasDatabase = !loading && !error && Boolean(database);

  return (
    <Card
      success={success}
      successTitle={false}
      onSuccessClose={() => setSuccess(undefined)}
      error={error}
      loading={loading}
      contentPadding={false}
      sx={{ width: "100%" }}
    >
      <CardHeader
        title="Database"
        subtitle="Attach a PostgreSQL database to your application"
      />
      {hasDatabase ? (
        <Box sx={{ p: 2 }}>
          <Typography>Database details will go here</Typography>
        </Box>
      ) : (
        <EmptyView onAttachClick={() => setIsAttachModalOpen(true)} />
      )}
      {hasDatabase && <CardFooter>&nbsp;</CardFooter>}
      {isAttachModalOpen && <>{/* Modal will go here */}</>}
    </Card>
  );
}
