import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface UpdateTeamProps {
  teamId: string;
  name: string;
}

export const updateTeam = ({
  name,
  teamId,
}: UpdateTeamProps): Promise<Team> => {
  return api.patch<Team>("/team", { name, teamId });
};

interface FetchTeamMembersProps {
  team?: Team;
}

export const useFetchTeamMembers = ({ team }: FetchTeamMembersProps) => {
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!team?.id || team.isDefault) {
      return;
    }

    setLoading(true);
    setError("");

    api
      .fetch<TeamMember[]>(`/team/members?teamId=${team.id}`)
      .then(members => {
        setTeamMembers(members);
      })
      .catch(() => {
        setError("Something went wrong while fetching team members.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [team]);

  return { teamMembers, loading, error };
};

interface InviteMemberProps {
  email: string;
  teamId: string;
  role: TeamRole;
}

export const inviteMember = ({ email, teamId, role }: InviteMemberProps) => {
  return api.post<{ token: string }>("/team/invite", { teamId, email, role });
};

interface InvitationAcceptProps {
  token?: string;
}

export const useInvitationAccept = ({ token }: InvitationAcceptProps) => {
  const [team, setTeam] = useState<Team>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!token) {
      return;
    }

    api
      .post<Team>("/team/enroll", { token })
      .then(team => {
        setTeam(team);
      })
      .catch(async res => {
        const data = await res.json();

        setError(
          data.error && data.error !== "unexpected-error"
            ? data.error
            : "Something went wrong while accepting the invitation."
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, [token]);

  return { team, loading, error };
};

interface RemoveMemberProps {
  teamId: string;
  memberId: string;
}

export const removeMember = ({ teamId, memberId }: RemoveMemberProps) => {
  return api.delete(`/team/member?teamId=${teamId}&memberId=${memberId}`);
};

interface RemoveTeamProps {
  teamId: string;
}

export const removeTeam = ({ teamId }: RemoveTeamProps) => {
  return api.delete(`/team?teamId=${teamId}`);
};
