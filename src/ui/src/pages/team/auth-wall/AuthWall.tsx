import { useContext, useState } from "react";
import { useSearchParams } from "react-router-dom";
import Box from "@mui/material/Box";
import Tab from "@mui/material/Tab";
import Tabs from "@mui/material/Tabs";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import EmptyPage from "~/components/EmptyPage";
import AuthWallEnvironments from "./AuthWallEnvironments";
import AuthWallLogins from "./AuthWallLogins";
import { useFetchTeamEnvs } from "./actions";

export default function AuthWall() {
  const { teams } = useContext(AuthContext);
  const { details } = useContext(RootContext);
  const team = useSelectedTeam({ teams });
  const [searchParams, setSearchParams] = useSearchParams();
  const [tab, setTab] = useState<"environments" | "logins">("environments");
  const [refreshToken, setRefreshToken] = useState<number>();
  const { apps, loading, error, paymentRequired } = useFetchTeamEnvs({
    teamId: team?.id,
    refreshToken,
  });

  if (details?.license?.edition === "community" || paymentRequired) {
    return <EmptyPage paymentRequired />;
  }

  if (!team) {
    return null;
  }

  const canManageLogins =
    team.currentUserRole === "owner" || team.currentUserRole === "admin";

  return (
    <Box sx={{ width: "100%" }} maxWidth="lg">
      {canManageLogins && (
        <Tabs
          value={tab}
          onChange={(_, value) => setTab(value)}
          textColor="inherit"
          indicatorColor="secondary"
          sx={{ mb: 2 }}
        >
          <Tab value="environments" label="Environments" />
          <Tab value="logins" label="Logins" />
        </Tabs>
      )}
      {tab === "environments" && (
        <AuthWallEnvironments
          team={team}
          apps={apps}
          loading={loading}
          error={error}
          appId={searchParams.get("appId")}
          onShowAllApps={() => setSearchParams({})}
          onUpdate={() => setRefreshToken(Date.now())}
          showTitle={!canManageLogins}
        />
      )}
      {tab === "logins" && canManageLogins && !loading && (
        <AuthWallLogins team={team} apps={apps} appsError={error} />
      )}
    </Box>
  );
}
