import { useContext, useState } from "react";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import { AuthContext } from "~/pages/auth/Auth.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import { deleteApp } from "../actions";

interface Props {
  app: App;
}

export default function FormDangerZone({ app }: Props) {
  const { teams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams, app });
  const slug = `/${team?.slug || ""}`;
  const [isConfirmModalOpen, toggleConfirmModal] = useState(false);

  return (
    <Card sx={{ mb: 4, borderColor: "rgba(255, 0, 0, 0.25)" }}>
      <CardHeader>
        <Typography variant="h2" sx={{ fontSize: 20, mb: 0.5 }} color="error">
          Danger zone
        </Typography>
        <Typography variant="subtitle2">
          Permanently delete your App and all of its contents from the Stormkit
          platform. This action cannot be undone — proceed with caution.
        </Typography>
      </CardHeader>
      <CardFooter>
        <Button
          color="secondary"
          variant="contained"
          type="submit"
          onClick={() => {
            toggleConfirmModal(true);
          }}
        >
          Remove application
        </Button>
      </CardFooter>
      {isConfirmModalOpen && (
        <ConfirmModal
          typeConfirmationText="permanently delete application"
          onCancel={() => {
            toggleConfirmModal(false);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            deleteApp({ app })
              .then(() => {
                setLoading(false);
                window.location.assign(slug);
              })
              .catch(() => {
                setLoading(false);
                setError(
                  "Something went wrong while deleting application. Please retry later."
                );
              });
          }}
        >
          This will completely remove the application. All associated files and
          endpoints will be gone. Remember there is no going back from here.
        </ConfirmModal>
      )}
    </Card>
  );
}
