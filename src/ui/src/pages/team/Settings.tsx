import { useContext } from "react";
import Box from "@mui/material/Box";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import Error404 from "~/components/Errors/Error404";
import TeamMembers from "./TeamMembers";
import TeamDelete from "./TeamDelete";
import TeamAPIKeys from "./TeamAPIKeys";
import TeamSettings from "./TeamSettings";

export default function Settings() {
  const { user, teams, reloadTeams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams });

  if (!team || team.isDefault || !teams) {
    return (
      <Box>
        <Error404 />
      </Box>
    );
  }

  return (
    <Box sx={{ width: "100%" }} maxWidth="md">
      <TeamSettings team={team} reloadTeams={reloadTeams} />
      <TeamMembers user={user!} reloadTeams={reloadTeams} team={team} />
      <TeamAPIKeys team={team} />
      <TeamDelete reloadTeams={reloadTeams} team={team} />
    </Box>
  );
}
