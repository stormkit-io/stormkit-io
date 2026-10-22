import type { AuthWallMode, TeamAuthWallApp } from "./actions";
import { useEffect, useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Link from "@mui/material/Link";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import LensIcon from "@mui/icons-material/Lens";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import { updateEnvAuthWall } from "./actions";

interface Props {
  team: Team;
  apps: TeamAuthWallApp[];
  loading: boolean;
  error?: string;

  // appId limits the list to one app, e.g. when linked from an environment.
  appId?: string | null;
  onShowAllApps: () => void;

  // onUpdate is called after environments are updated, to reload them.
  onUpdate: () => void;

  // showTitle shows the card's title; it is hidden when a tab names it.
  showTitle?: boolean;
}

export const modeLabels: Record<string, string> = {
  "": "Off",
  dev: "Deployment previews only",
  all: "All URLs, including domains",
};

const subtitle = "Select environments to choose whether they require a login.";

const environmentsLabel = (count: number) =>
  count === 1 ? "1 environment" : `${count} environments`;

export default function AuthWallEnvironments({
  team,
  apps,
  loading,
  error,
  appId,
  onShowAllApps,
  onUpdate,
  showTitle = true,
}: Props) {
  const [modes, setModes] = useState<Record<string, AuthWallMode>>({});
  const [saveError, setSaveError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [selected, setSelected] = useState<string[]>([]);
  const [bulkMode, setBulkMode] = useState<AuthWallMode>("all");
  const [applying, setApplying] = useState(false);

  useEffect(() => {
    const initial: Record<string, AuthWallMode> = {};

    apps.forEach(app =>
      app.environments.forEach(env => {
        initial[env.id] = env.authWall;
      }),
    );

    setModes(initial);
  }, [apps]);

  const visibleApps = useMemo(
    () => (appId ? apps.filter(app => app.id === appId) : apps),
    [apps, appId],
  );

  const visibleEnvIds = useMemo(
    () => visibleApps.flatMap(app => app.environments.map(env => env.id)),
    [visibleApps],
  );

  // Environments of other apps are never changed while one app is shown.
  useEffect(() => {
    setSelected(selected => selected.filter(id => visibleEnvIds.includes(id)));
  }, [visibleEnvIds]);

  const allSelected =
    visibleEnvIds.length > 0 && selected.length === visibleEnvIds.length;

  const toggle = (envId: string) => {
    setSelected(
      selected.includes(envId)
        ? selected.filter(id => id !== envId)
        : [...selected, envId],
    );
  };

  const resetMessages = () => {
    setSaveError(undefined);
    setSuccess(undefined);
  };

  const applyToSelected = async () => {
    resetMessages();
    setApplying(true);

    const envIds = [...selected];
    const results = await Promise.allSettled(
      envIds.map(envId =>
        updateEnvAuthWall({ teamId: team.id, envId, authWall: bulkMode }),
      ),
    );

    const updated = envIds.filter((_, i) => results[i].status === "fulfilled");
    const failed = envIds.filter((_, i) => results[i].status === "rejected");

    setModes(modes => ({
      ...modes,
      ...Object.fromEntries(updated.map(id => [id, bulkMode])),
    }));

    setSelected(failed);
    setApplying(false);

    if (updated.length > 0) {
      onUpdate();
    }

    if (failed.length > 0) {
      setSaveError(
        `Could not update ${environmentsLabel(failed.length)}. ${failed.length === 1 ? "It is" : "They are"} still selected so you can retry.`,
      );
    } else {
      setSuccess(
        `Auth Wall of ${environmentsLabel(updated.length)} set to: ${modeLabels[bulkMode]}.`,
      );
    }
  };

  return (
    <Card
      id="environments"
      sx={{ width: "100%" }}
      loading={loading}
      error={error || saveError}
      success={success}
      onSuccessClose={() => setSuccess(undefined)}
      onErrorClose={() => setSaveError(undefined)}
      info={
        !loading && !error && !visibleApps.length && "This team has no apps."
      }
    >
      <CardHeader
        title={showTitle ? "Environments" : undefined}
        subtitle={showTitle ? subtitle : undefined}
        actions={
          appId && (
            <Link component="button" onClick={onShowAllApps}>
              Show all apps
            </Link>
          )
        }
      >
        {!showTitle && (
          <Typography color="text.secondary">{subtitle}</Typography>
        )}
      </CardHeader>
      {/* Card places its alerts right after the content, without a gap. */}
      <Box sx={{ mb: error || saveError || success ? 4 : 0 }}>
        {selected.length > 0 && (
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 2,
              px: 2,
              py: 1,
              mb: 2,
              bgcolor: "container.paper",
              borderRadius: 1,
            }}
          >
            <Typography sx={{ flex: 1 }}>
              {environmentsLabel(selected.length)} selected
            </Typography>
            <Typography color="text.secondary">Set Auth Wall to</Typography>
            <Select
              size="small"
              variant="outlined"
              displayEmpty
              value={bulkMode}
              inputProps={{ "aria-label": "Auth Wall of selected" }}
              sx={{ minWidth: 240 }}
              onChange={e => setBulkMode(e.target.value as AuthWallMode)}
            >
              {Object.entries(modeLabels).map(([value, text]) => (
                <MenuItem key={value} value={value}>
                  {text}
                </MenuItem>
              ))}
            </Select>
            <Button
              variant="contained"
              color="secondary"
              loading={applying}
              onClick={applyToSelected}
            >
              Apply
            </Button>
          </Box>
        )}
        {visibleApps.length > 0 && (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox">
                  <Checkbox
                    size="small"
                    color="secondary"
                    checked={allSelected}
                    indeterminate={selected.length > 0 && !allSelected}
                    onChange={() =>
                      setSelected(allSelected ? [] : visibleEnvIds)
                    }
                    slotProps={{
                      input: { "aria-label": "Select all environments" },
                    }}
                  />
                </TableCell>
                <TableCell>App</TableCell>
                <TableCell>Environment</TableCell>
                <TableCell sx={{ width: 280 }}>Auth Wall</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {visibleApps.map(app =>
                app.environments.map((env, i) => (
                  <TableRow
                    key={env.id}
                    hover
                    selected={selected.includes(env.id)}
                    onClick={() => toggle(env.id)}
                    sx={{ cursor: "pointer" }}
                  >
                    <TableCell padding="checkbox">
                      <Checkbox
                        size="small"
                        color="secondary"
                        checked={selected.includes(env.id)}
                        slotProps={{
                          // The row handles the click.
                          input: {
                            "aria-label": `Select ${app.displayName} ${env.name}`,
                            readOnly: true,
                          },
                        }}
                      />
                    </TableCell>
                    {i === 0 && (
                      <TableCell
                        rowSpan={app.environments.length}
                        sx={{ verticalAlign: "middle", fontWeight: "bold" }}
                      >
                        {app.displayName}
                      </TableCell>
                    )}
                    <TableCell>{env.name}</TableCell>
                    <TableCell>
                      <Box
                        component="span"
                        sx={{
                          display: "inline-flex",
                          alignItems: "center",
                          gap: 1,
                          color: modes[env.id]
                            ? "text.primary"
                            : "text.secondary",
                        }}
                      >
                        <LensIcon
                          sx={{ fontSize: 8 }}
                          color={modes[env.id] ? "success" : "disabled"}
                        />
                        {modeLabels[modes[env.id] ?? env.authWall]}
                      </Box>
                    </TableCell>
                  </TableRow>
                )),
              )}
            </TableBody>
          </Table>
        )}
      </Box>
    </Card>
  );
}
