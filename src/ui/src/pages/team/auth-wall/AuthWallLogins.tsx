import type { TeamAuthWallApp, TeamAuthWallLogin } from "./actions";
import { useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import EditIcon from "@mui/icons-material/Edit";
import PersonAddIcon from "@mui/icons-material/PersonAdd";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import { formatDate } from "~/utils/helpers/date";
import AuthWallLoginModal from "./AuthWallLoginModal";
import { deleteTeamLogins, useFetchTeamLogins } from "./actions";

interface Props {
  team: Team;
  apps: TeamAuthWallApp[];

  // appsError is set when the team's apps could not be loaded. Logins cannot
  // be given access to apps without them.
  appsError?: string;
}

export default function AuthWallLogins({ team, apps, appsError }: Props) {
  const [refreshToken, setRefreshToken] = useState<number>();
  const [checked, setChecked] = useState<string[]>([]);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [modal, setModal] = useState<{ login?: TeamAuthWallLogin }>();
  const { loading, error, logins, hasNextPage, loadMore } = useFetchTeamLogins({
    teamId: team.id,
    refreshToken,
  });

  const appNames = useMemo(
    () => Object.fromEntries(apps.map(app => [app.id, app.displayName])),
    [apps],
  );

  const describeApps = (login: TeamAuthWallLogin) =>
    login.allApps
      ? "All apps"
      : login.appIds.map(id => appNames[id] || id).join(", ");

  const toggle = (id: string) => {
    setChecked(
      checked.includes(id) ? checked.filter(c => c !== id) : [...checked, id],
    );
  };

  return (
    <>
      <Card
        id="logins"
        sx={{ width: "100%" }}
        loading={loading && logins.length === 0}
        error={appsError || error}
        info={
          !loading &&
          !error &&
          !logins.length &&
          "This team has no logins yet. Click the button above to add one."
        }
      >
        <CardHeader
          actions={
            <IconButton
              aria-label="New login"
              disabled={!!appsError}
              onClick={() => setModal({})}
            >
              <PersonAddIcon />
            </IconButton>
          }
        >
          <Typography color="text.secondary">
            People who can pass the Auth Wall of this team&apos;s apps. Removing
            a login or its access takes effect when the visitor&apos;s session
            expires (at most 24 hours).
          </Typography>
        </CardHeader>
        <Box>
          {logins.length > 0 && (
            <CardRow sx={{ backgroundColor: "container.paper" }}>
              <Box
                sx={{ display: "flex", width: "100%" }}
                color="text.secondary"
              >
                <Typography sx={{ flex: 1 }}>Email</Typography>
                <Typography sx={{ flex: 1 }}>Apps</Typography>
                <Typography sx={{ width: 180 }}>Last login</Typography>
                <Box sx={{ width: 40 }} />
              </Box>
            </CardRow>
          )}
          {logins.map(login => (
            <CardRow key={login.id}>
              <Box
                sx={{ display: "flex", width: "100%", alignItems: "center" }}
              >
                <FormControlLabel
                  label={login.email}
                  sx={{ flex: 1 }}
                  control={
                    <Checkbox
                      size="small"
                      color="secondary"
                      checked={checked.includes(login.id)}
                      onChange={() => toggle(login.id)}
                    />
                  }
                />
                <Typography sx={{ flex: 1 }}>{describeApps(login)}</Typography>
                <Typography sx={{ width: 180 }}>
                  {login.lastLogin ? formatDate(login.lastLogin) : "Never"}
                </Typography>
                <IconButton
                  aria-label={`Edit ${login.email}`}
                  onClick={() => setModal({ login })}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
              </Box>
            </CardRow>
          ))}
        </Box>
        {logins.length > 0 && (
          <CardFooter>
            <Box sx={{ display: "flex", justifyContent: "space-between" }}>
              <Button
                variant="contained"
                color="secondary"
                disabled={checked.length === 0}
                onClick={() => setShowDeleteModal(true)}
              >
                Remove selected
              </Button>
              {hasNextPage && (
                <Button variant="text" loading={loading} onClick={loadMore}>
                  Load more
                </Button>
              )}
            </Box>
          </CardFooter>
        )}
      </Card>
      {modal && (
        <AuthWallLoginModal
          team={team}
          apps={apps}
          login={modal.login}
          onSuccess={() => setRefreshToken(Date.now())}
          onClose={() => setModal(undefined)}
        />
      )}
      {showDeleteModal && (
        <ConfirmModal
          onCancel={() => setShowDeleteModal(false)}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);

            deleteTeamLogins({ teamId: team.id, loginIds: checked })
              .then(() => {
                setRefreshToken(Date.now());
                setShowDeleteModal(false);
                setChecked([]);
              })
              .catch(() => {
                setError("Something went wrong while removing the logins.");
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          You are about to remove{" "}
          {checked.length === 1 ? "1 login" : `${checked.length} logins`} from
          every app of this team. Visitors already logged in keep access until
          their session expires (at most 24 hours).
        </ConfirmModal>
      )}
    </>
  );
}
