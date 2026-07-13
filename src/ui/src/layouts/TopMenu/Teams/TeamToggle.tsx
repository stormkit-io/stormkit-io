import { useState, useContext } from "react";
import Box from "@mui/material/Box";
import SwitchRight from "@mui/icons-material/SwitchRight";
import SwitchLeft from "@mui/icons-material/SwitchLeft";
import Button from "@mui/material/Button";
import Tooltip from "@mui/material/Tooltip";
import { AuthContext } from "~/pages/auth/Auth.context";
import TeamModal from "./TeamModal";
import TeamMenu from "./TeamMenu";
import { useSelectedTeam } from "./actions";

const PERSONAL_TEAM = "Personal";

interface Props {
  app?: App;
}

export default function TeamsToggle({ app }: Props) {
  const { teams, reloadTeams } = useContext(AuthContext);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const selectedTeam = useSelectedTeam({ teams, app });
  const Switch = isMenuOpen ? SwitchRight : SwitchLeft;

  return (
    <Box sx={{ display: "flex", alignItems: "center" }}>
      <Tooltip
        arrow
        open={isMenuOpen}
        placement="bottom-start"
        slotProps={{
          popper: {
            modifiers: [{ name: "offset", options: { offset: [0, 12] } }],
          },
        }}
        title={
          <Box sx={{ p: 1 }}>
            <TeamMenu
              selectedTeam={selectedTeam}
              teams={teams}
              onClickAway={() => {
                setIsMenuOpen(false);
              }}
              onSettingsClick={() => {
                setIsMenuOpen(false);
              }}
              onCreateTeamButtonClicked={() => {
                setIsModalOpen(true);
                setIsMenuOpen(false);
              }}
            />
          </Box>
        }
      >
        <Box>
          <Button
            variant="text"
            type="button"
            onClick={e => {
              e.preventDefault();
              setIsMenuOpen(!isMenuOpen);
            }}
          >
            {selectedTeam?.isDefault
              ? PERSONAL_TEAM
              : selectedTeam?.name || "Select a team"}
            <Switch sx={{ transform: "rotate(90deg)", ml: 1, fontSize: 16 }} />
          </Button>
        </Box>
      </Tooltip>

      {isModalOpen && (
        <TeamModal
          reload={reloadTeams}
          onClose={() => {
            setIsModalOpen(false);
          }}
        />
      )}
    </Box>
  );
}
