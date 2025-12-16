import { useContext, useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import AddIcon from "@mui/icons-material/Add";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import TextField from "@mui/material/TextField";
import Switch from "@mui/material/Switch";
import FormControlLabel from "@mui/material/FormControlLabel";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import EmptyPage from "~/components/EmptyPage";
import CardFooter from "~/components/CardFooter";
import { useFetchSchema, createSchema, updateSchemaConfig } from "./actions";

interface EmptyViewProps {
  onAttachClick: () => void;
  isAttachLoading?: boolean;
}

function EmptyView({ onAttachClick, isAttachLoading }: EmptyViewProps) {
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
          loading={isAttachLoading}
        >
          Attach Database
        </Button>
      </Box>
    </EmptyPage>
  );
}

export default function Database() {
  const { environment } = useContext(EnvironmentContext);
  const [refreshToken, setRefreshToken] = useState<number>();
  const result = useFetchSchema({ envId: environment.id!, refreshToken });
  const [updating, setUpdating] = useState(false);
  const [success, setSuccess] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [isAttaching, setIsAttaching] = useState(false);
  const [migrationsEnabled, setMigrationsEnabled] = useState(false);
  const { schema, loading, error } = result;
  const hasSchema = !loading && !error && Boolean(schema);

  useEffect(() => {
    if (schema !== null) {
      setMigrationsEnabled(Boolean(schema.migrationsEnabled));
    }
  }, [schema?.migrationsEnabled]);

  const handleAttachSchema = async () => {
    setIsAttaching(true);

    try {
      await createSchema({
        appId: environment.appId!,
        envId: environment.id!,
      });

      setFormError(undefined);
      setSuccess("Schema attached successfully");
      setRefreshToken(Date.now());
    } catch (e) {
      setSuccess(undefined);
      setFormError("Unknown error while attaching schema. Please try again.");
    } finally {
      setIsAttaching(false);
    }
  };

  const handleSubmit: React.FormEventHandler = async (
    e: React.FormEvent<HTMLFormElement>
  ) => {
    e.preventDefault();

    const formData = new FormData(e.currentTarget);
    const migrationsFolder = formData.get("migrationsFolder") as string;

    setUpdating(true);
    setFormError(undefined);
    setSuccess(undefined);

    try {
      await updateSchemaConfig({
        appId: environment.appId!,
        envId: environment.id!,
        migrationsFolder,
        migrationsEnabled,
      });

      setSuccess("Schema updated successfully");
    } catch (e) {
      setFormError("Unknown error while updating schema. Please try again.");
    } finally {
      setUpdating(false);
    }
  };

  return (
    <Card
      component="form"
      success={success}
      successTitle={false}
      onSuccessClose={() => setSuccess(undefined)}
      error={error || formError}
      loading={loading}
      sx={{ width: "100%" }}
      onSubmit={handleSubmit}
    >
      <CardHeader
        title="Database"
        subtitle="Attach and access a PostgreSQL schema to manage your application data"
      />
      {hasSchema ? (
        <>
          <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
            <FormControlLabel
              sx={{ pl: 0, ml: 0 }}
              label="Enable schema migrations"
              control={
                <Switch
                  name="migrationsEnabled"
                  color="secondary"
                  checked={migrationsEnabled}
                  onChange={e => {
                    setMigrationsEnabled(e.target.checked);
                  }}
                />
              }
              labelPlacement="start"
            />
            <Typography color="text.secondary" variant="body2">
              When applied, Stormkit will automatically migrate your database
              schema based on the migration files in your repository.
            </Typography>
          </Box>
          <Box sx={{ mb: 4 }}>
            <TextField
              label="Migrations path"
              name="migrationsFolder"
              placeholder="/migrations"
              fullWidth
              defaultValue={schema?.migrationsFolder || ""}
              variant="filled"
              autoComplete="off"
              helperText="Path to the folder containing your database migration files."
              slotProps={{
                inputLabel: {
                  shrink: true,
                },
              }}
            />
          </Box>
        </>
      ) : (
        <EmptyView
          onAttachClick={handleAttachSchema}
          isAttachLoading={isAttaching}
        />
      )}
      {hasSchema && (
        <CardFooter>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={updating}
          >
            Save
          </Button>
        </CardFooter>
      )}
    </Card>
  );
}
