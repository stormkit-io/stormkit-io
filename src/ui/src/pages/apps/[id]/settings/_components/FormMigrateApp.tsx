import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Select from "@mui/material/Select";
import InputLabel from "@mui/material/InputLabel";
import FormControl from "@mui/material/FormControl";
import Typography from "@mui/material/Typography";
import MenuItem from "@mui/material/MenuItem";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import api from "~/utils/api/Api";
import { RootContext } from "~/pages/Root.context";

interface Props {
  app: App;
  teams: Team[];
}

export default function FormMigrateApp({ app, teams }: Props) {
  const { details } = useContext(RootContext);
  const [isConfirmModalOpen, toggleConfirmModal] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<Team | void>(
    teams?.find(t => t.id === app.teamId)
  );

  const hasWriteAccess = (t: Team) =>
    t.currentUserRole === "admin" || t.currentUserRole == "owner";

  if (details?.license?.edition === "community") {
    return <></>;
  }

  return (
    <Card sx={{ mb: 4 }}>
      <CardHeader
        title="Migrate app"
        subtitle="Move this application to a different team. At least 'Admin' role in destination team is required."
      />
      <Box sx={{ mb: 4 }}>
        <FormControl variant="standard" fullWidth>
          <InputLabel id="migrate-app-dropdown" sx={{ pl: 2, pt: 1 }}>
            Select a team to migrate
          </InputLabel>
          <Select
            labelId="migrate-app-dropdown"
            variant="filled"
            name="migrate app"
            value={selectedTeam?.id || teams?.[0].id}
            onChange={e =>
              setSelectedTeam(teams?.find(t => t.id === e.target.value))
            }
            sx={{ minWidth: 150 }}
          >
            {teams
              ?.filter(t => hasWriteAccess(t))
              .map(team => (
                <MenuItem value={team.id} key={team.id}>
                  {team.name}
                </MenuItem>
              ))}
          </Select>
        </FormControl>
      </Box>
      <CardFooter>
        <Button
          color="secondary"
          variant="contained"
          type="submit"
          onClick={() => {
            toggleConfirmModal(true);
          }}
        >
          Move application
        </Button>
      </CardFooter>
      {isConfirmModalOpen && selectedTeam ? (
        <ConfirmModal
          typeConfirmationText={"migrate"}
          onCancel={() => {
            toggleConfirmModal(false);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);

            api
              .post("/team/migrate", {
                appId: app.id,
                teamId: selectedTeam.id,
              })
              .then(() => {
                setLoading(false);
                window.location.reload();
              })
              .catch(() => {
                setLoading(false);
                setError(
                  "Something went wrong while migrating application. Please retry later."
                );
              });
          }}
        >
          This will migrate the{" "}
          <Typography component="span" color="text.secondary" fontSize="medium">
            {app.displayName}
          </Typography>{" "}
          application to{" "}
          <Typography component="span" color="text.secondary" fontSize="medium">
            {selectedTeam.name}
          </Typography>{" "}
          team. Some users may no longer access this application.
        </ConfirmModal>
      ) : (
        ""
      )}
    </Card>
  );
}
