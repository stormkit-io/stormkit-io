import React from "react";
import { LocalStorage } from "~/utils/storage";
import XIcon from "@mui/icons-material/X";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import DiscordIcon from "~/assets/logos/discord.svg";

interface Props {
  isOpen: boolean;
  welcomeModalId: string;
  toggleModal: (val: boolean) => void;
}

const Welcome: React.FC<Props> = ({
  isOpen,
  toggleModal,
  welcomeModalId,
}): React.ReactElement => {
  const close = () => {
    toggleModal(false);
    LocalStorage.set(welcomeModalId, "shown");
  };

  return (
    <Modal open={isOpen} onClose={close}>
      <Card>
        <CardHeader title="Welcome to Stormkit 🎉" />
        <Button
          type="button"
          href="https://discord.gg/6yQWhyY"
          target="_blank"
          rel="noreferrer"
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            boxShadow: 1,
            backgroundColor: "container.transparent",
            p: { xs: 2, md: 4 },
            mb: 4,
          }}
        >
          <Box
            component="img"
            src={DiscordIcon}
            alt="Discord"
            sx={{ color: "#5865F2", width: 32, mr: 4 }}
          />
          <Typography
            component="span"
            sx={{ display: "flex", flexDirection: "column", flexGrow: 1 }}
          >
            <Box component="b">Join our Discord community</Box>
            <Box component="span">Ask questions and join discussions.</Box>
          </Typography>
          <ChevronRightIcon />
        </Button>
        <Button
          type="button"
          href="https://x.com/stormkitio"
          target="_blank"
          rel="noreferrer"
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            boxShadow: 1,
            backgroundColor: "container.transparent",
            p: { xs: 2, md: 4 },
            mb: 4,
          }}
        >
          <XIcon sx={{ fontSize: 32, mr: 4 }} />
          <Typography
            sx={{ display: "flex", flexDirection: "column", flexGrow: 1 }}
          >
            <Box component="b">Follow us on X</Box>
            <Box component="span">
              Stay tuned about latest developments on Stormkit.
            </Box>
          </Typography>
          <ChevronRightIcon />
        </Button>
        <CardFooter>
          <Button
            type="button"
            variant="contained"
            color="secondary"
            onClick={close}
          >
            Close
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
};

export default Welcome;
