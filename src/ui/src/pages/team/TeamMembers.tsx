import { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import InviteMemberModal from "./InviteMemberModal";
import { useFetchTeamMembers, removeMember } from "./actions";

interface Props {
  user: User;
  team: Team;
  reloadTeams?: () => void;
}

export default function TeamMembers({ user, team, reloadTeams }: Props) {
  const [isNewMemberModalOpen, setIsNewMemberModalOpen] = useState(false);
  const tmFetchResult = useFetchTeamMembers({ team });
  const [memberToBeRemoved, setMemberToBeRemoved] = useState<TeamMember>();
  const hasWriteAccess =
    team.currentUserRole === "owner" || team.currentUserRole === "admin";

  const { teamMembers, error: tmError, loading } = tmFetchResult;

  return (
    <>
      <Card
        error={tmError}
        sx={{ mb: 2 }}
        loading={loading}
        contentPadding={false}
      >
        <CardHeader
          title="Team members"
          subtitle="Invite team members to collaborate on your projects."
        />
        <Box>
          {teamMembers?.map(member => (
            <CardRow
              key={member.userId}
              chipLabel={
                <Typography component="span">{member.role}</Typography>
              }
              menuLabel={`Member ${member.id} menu`}
              menuItems={
                hasWriteAccess || user.id === member.userId
                  ? [
                      {
                        text: member.userId === user?.id ? "Leave" : "Remove",
                        onClick: () => {
                          setMemberToBeRemoved(member);
                        },
                      },
                    ]
                  : []
              }
            >
              <Typography>
                {member.fullName?.trim() || member.displayName}
              </Typography>
              <Typography sx={{ color: "text.secondary" }}>
                {member.email}
              </Typography>
            </CardRow>
          ))}
        </Box>
        {hasWriteAccess && (
          <CardFooter>
            <Button
              variant="contained"
              color="secondary"
              onClick={e => {
                e.preventDefault();
                setIsNewMemberModalOpen(true);
              }}
            >
              Invite member
            </Button>
          </CardFooter>
        )}
      </Card>
      {isNewMemberModalOpen && (
        <InviteMemberModal
          teamId={team.id}
          onClose={() => {
            setIsNewMemberModalOpen(false);
          }}
        />
      )}
      {memberToBeRemoved && (
        <ConfirmModal
          onCancel={() => {
            setMemberToBeRemoved(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            removeMember({
              memberId: memberToBeRemoved.id,
              teamId: team.id,
            })
              .then(() => {
                reloadTeams?.();
              })
              .catch(async res => {
                const data = await res.json();

                setError(
                  data.error || "Something went wrong while removing member."
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          <>
            {memberToBeRemoved.userId === user.id ? (
              <Box component="span">
                You are about to leave the team. You will need another
                invitation to re-join.
              </Box>
            ) : (
              <Box component="span">
                You are about to remove{" "}
                <Box component="span" sx={{ fontWeight: "bold", opacity: 0.5 }}>
                  {memberToBeRemoved.email}
                </Box>{" "}
                from the team.
                <br /> They will lose access to all apps under this team.
              </Box>
            )}
          </>
        </ConfirmModal>
      )}
    </>
  );
}
