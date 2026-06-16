import { useMemo } from "react";
import { useParams } from "react-router";
import api from "~/utils/api/Api";

interface CreateTeamProps {
  name: string;
}

export const createTeam = ({ name }: CreateTeamProps): Promise<Team> => {
  return api.post<{ team: Team }>("/v1/teams", { name }).then(res => res.team);
};

interface UseSelectedTeamProps {
  teams?: Team[];
  app?: App;
}

export const useSelectedTeam = ({
  teams,
  app,
}: UseSelectedTeamProps): Team | undefined => {
  const params = useParams();
  const findDefaultTeam = () => teams?.find(t => t.isDefault);

  const team = useMemo(() => {
    if (params.team) {
      return teams?.find(t => t.slug === params.team || t.id === params.team);
    }

    if (app?.teamId) {
      return teams?.find(t => t.id === app.teamId);
    }

    const teamId = localStorage.getItem("teamId");

    if (teamId) {
      return teams?.find(t => t.id === teamId) || findDefaultTeam();
    }

    return findDefaultTeam();
  }, [teams, params, app]);

  if (team) {
    localStorage.setItem("teamId", team.id);
  }

  return team;
};
