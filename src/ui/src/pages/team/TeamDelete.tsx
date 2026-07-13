import { useState } from "react";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import { removeTeam } from "./actions";

interface Props {
  team: Team;
  reloadTeams?: () => void;
}

export default function TeamDelete({ team, reloadTeams }: Props) {
  const [confirmModal, setConfirmModal] = useState(false);
  const hasWriteAccess =
    team.currentUserRole === "owner" || team.currentUserRole === "admin";

  if (!hasWriteAccess) {
    return <></>;
  }

  return (
    <>
      <Card sx={{ mb: 2, borderColor: "rgba(255, 0, 0, 0.25)" }}>
        <CardHeader>
          <Typography variant="h2" sx={{ fontSize: 20, mb: 0.5 }} color="error">
            Danger zone
          </Typography>
          <Typography variant="subtitle2">
            Permanently delete your Team and all of its contents from the
            Stormkit platform. This action cannot be undone — proceed with
            caution.
          </Typography>
        </CardHeader>

        <CardFooter>
          <Button
            variant="contained"
            color="secondary"
            onClick={e => {
              e.preventDefault();
              setConfirmModal(true);
            }}
          >
            Delete
          </Button>
        </CardFooter>
      </Card>
      {confirmModal && (
        <ConfirmModal
          typeConfirmationText="Permanently delete my team"
          onCancel={() => {
            setConfirmModal(false);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            removeTeam({
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
          This will delete your team and all of its contents. This action cannot
          be undone.
        </ConfirmModal>
      )}
    </>
  );
}
