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
import { useFetchSchema } from "./actions";

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
  const result = useFetchSchema({ envId: environment.id!, refreshToken });
  const [success, setSuccess] = useState<string>();
  const [isAttachModalOpen, setIsAttachModalOpen] = useState(false);
  const { schema, loading, error } = result;
  const hasSchema = !loading && !error && Boolean(schema);

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
        subtitle="Attach a PostgreSQL schema to manage your application data"
      />
      {hasSchema ? (
        <Box sx={{ p: 2 }}>{/* Database schema details will go here */}</Box>
      ) : (
        <EmptyView onAttachClick={() => setIsAttachModalOpen(true)} />
      )}
      {hasSchema && <CardFooter>&nbsp;</CardFooter>}
      {isAttachModalOpen && <>{/* Modal will go here */}</>}
    </Card>
  );
}
