import { useContext, useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import AddIcon from "@mui/icons-material/Add";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import TextField from "@mui/material/TextField";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import EmptyPage from "~/components/EmptyPage";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import Help from "~/components/Help";
import { Switch } from "~/components/Form";
import * as actions from "./actions";
import { RootContext } from "~/pages/Root.context";

const { createSchema, deleteSchema, updateSchema, useFetchSchema } = actions;

interface EmptyViewProps {
  onAttachClick: () => void;
  isAttachLoading?: boolean;
  isCloud?: boolean;
}

function EmptyView({
  onAttachClick,
  isAttachLoading,
  isCloud,
}: EmptyViewProps) {
  return (
    <EmptyPage>
      <Typography
        component="span"
        variant="h6"
        sx={{ mb: 4, display: "block" }}
      >
        {isCloud
          ? "The database feature is currently only available for self-hosted installations."
          : "No database attached to this environment"}
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
        {!isCloud && (
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
        )}
      </Box>
    </EmptyPage>
  );
}

export default function Database() {
  const { details } = useContext(RootContext);
  const isCloud = details?.stormkit?.edition === "cloud";
  const { environment: env } = useContext(EnvironmentContext);
  const [refreshToken, setRefreshToken] = useState<number>();
  const result = useFetchSchema({ envId: env.id!, refreshToken, isCloud });
  const [updating, setUpdating] = useState(false);
  const [success, setSuccess] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [isAttaching, setIsAttaching] = useState(false);
  const [migrationsEnabled, setMigrationsEnabled] = useState(false);
  const [injectEnvVars, setInjectEnvVars] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(false);
  const { schema, loading, error } = result;
  const hasSchema = !loading && !error && Boolean(schema);

  // Sync props with form state when schema is loaded
  useEffect(() => {
    if (schema !== null) {
      setMigrationsEnabled(Boolean(schema.migrationsEnabled));
      setInjectEnvVars(Boolean(schema.injectEnvVars));
    }
  }, [schema?.migrationsEnabled, schema?.injectEnvVars]);

  const handleAttachSchema = async () => {
    setIsAttaching(true);

    try {
      await createSchema({
        appId: env.appId!,
        envId: env.id!,
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
    e: React.FormEvent<HTMLFormElement>,
  ) => {
    e.preventDefault();

    const formData = new FormData(e.currentTarget);
    const migrationsFolder = formData.get("migrationsFolder") as string;

    setUpdating(true);
    setFormError(undefined);
    setSuccess(undefined);

    try {
      await updateSchema({
        appId: env.appId!,
        envId: env.id!,
        migrationsFolder,
        migrationsEnabled,
        injectEnvVars,
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
      {hasSchema && !isCloud ? (
        <>
          <Switch
            name="migrationsEnabled"
            label="Enable schema migrations"
            description="When applied, Stormkit will automatically migrate your database schema based on the migration files in your repository."
            checked={migrationsEnabled}
            setChecked={value => {
              setMigrationsEnabled(value);
            }}
          />

          <Switch
            name="injectEnvVars"
            label="Inject environment variables"
            checked={injectEnvVars}
            setChecked={value => {
              setInjectEnvVars(value);
            }}
            description={
              <>
                When enabled, Stormkit will make environment variables available
                to your deployment.{" "}
                <Help
                  title="How does this work?"
                  buttonVariant="link"
                  buttonText="Learn more."
                >
                  <Box>
                    <Typography>
                      When this option is enabled, Stormkit will inject the
                      following environment variables at build time and make
                      them available at runtime:
                    </Typography>
                    <Box component="ul" sx={{ my: 2 }}>
                      {[
                        "POSTGRES_HOST",
                        "POSTGRES_PORT",
                        "POSTGRES_DB",
                        "POSTGRES_SCHEMA",
                        "POSTGRES_USER",
                        "POSTGRES_PASSWORD",
                        "DATABASE_URL",
                      ].map(varName => (
                        <Box component="li" key={varName}>
                          <Typography
                            component="span"
                            sx={{ fontFamily: "monospace" }}
                          >
                            - {varName}
                          </Typography>
                        </Box>
                      ))}
                    </Box>
                    <Typography>
                      The <Box component="code">DATABASE_URL</Box> is a
                      connection string that contains all the necessary
                      information to connect to your database. Example:
                    </Typography>
                    <Box
                      component="code"
                      sx={{
                        mt: 2,
                        p: 2,
                        display: "block",
                        overflowX: "auto",
                      }}
                    >
                      {`postgresql://user:password@host:port/dbname?options=-csearch_path=schema_name`}
                    </Box>
                  </Box>
                </Help>
              </>
            }
          />

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
          isCloud={isCloud}
          onAttachClick={handleAttachSchema}
          isAttachLoading={isAttaching}
        />
      )}
      {hasSchema && (
        <CardFooter>
          <Button
            variant="text"
            color="primary"
            type="button"
            onClick={() => setDeleteConfirm(true)}
            sx={{ mr: 2 }}
          >
            Delete
          </Button>
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
      {deleteConfirm && (
        <ConfirmModal
          title="Delete Database Schema"
          onConfirm={async ({ setLoading, setError }) => {
            setLoading(true);

            deleteSchema({ appId: env.appId!, envId: env.id! })
              .then(() => {
                setDeleteConfirm(false);
                setSuccess("Schema deleted successfully");
                setRefreshToken(Date.now());
              })
              .catch(res => {
                if (res.status === 403) {
                  setError(
                    "You don't have permission to delete this database schema. Contact team admin.",
                  );
                } else {
                  setError(
                    "Failed to delete database schema. Please try again.",
                  );
                }
              })
              .finally(() => {
                setLoading(false);
              });
          }}
          onCancel={() => setDeleteConfirm(false)}
        >
          You are about to permanently delete this environment's database
          schema. All application data will be lost and this action cannot be
          undone.
        </ConfirmModal>
      )}
    </Card>
  );
}
