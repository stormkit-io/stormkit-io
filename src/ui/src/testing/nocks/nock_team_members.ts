import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchTeamMembersProps {
  teamId: string;
  status?: number;
  response: TeamMember[];
}

export const mockFetchTeamMembers = ({
  teamId,
  status = 200,
  response,
}: MockFetchTeamMembersProps) =>
  nock(endpoint).get(`/team/members?teamId=${teamId}`).reply(status, response);

interface MockRemoveTeamMemberProps {
  teamId: string;
  memberId: string;
  status?: number;
  response?: { ok: boolean };
}

export const mockRemoveTeamMember = ({
  teamId,
  memberId,
  status = 200,
  response = { ok: true },
}: MockRemoveTeamMemberProps) =>
  nock(endpoint)
    .delete(`/team/member?teamId=${teamId}&memberId=${memberId}`)
    .reply(status, response);

interface MockInviteTeamMemberProps {
  teamId: string;
  email: string;
  role: TeamRole;
  status?: number;
  response: { token: string };
}

export const mockInviteTeamMember = ({
  teamId,
  email,
  role,
  status = 200,
  response = { token: "my-token" },
}: MockInviteTeamMemberProps) =>
  nock(endpoint)
    .post("/team/invite", { teamId, email, role })
    .reply(status, response);

interface MockInvitationAcceptProps {
  token: string;
  status?: number;
  response: Team;
}

export const mockInvitationAccept = ({
  token,
  status = 200,
  response,
}: MockInvitationAcceptProps) =>
  nock(endpoint).post("/team/enroll", { token }).reply(status, response);
