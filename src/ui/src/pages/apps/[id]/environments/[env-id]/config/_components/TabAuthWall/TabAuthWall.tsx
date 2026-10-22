import { useContext } from "react";
import { Navigate } from "react-router-dom";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";

interface Props {
  app: App;
}

// TabAuthWall sends old links to the environment's Auth Wall tab to the team
// page, where Auth Wall is now managed.
export default function TabAuthWall({ app }: Props) {
  const { teams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams, app });

  if (!team) {
    return null;
  }

  return <Navigate to={`/${team.slug}/auth-wall?appId=${app.id}`} replace />;
}
