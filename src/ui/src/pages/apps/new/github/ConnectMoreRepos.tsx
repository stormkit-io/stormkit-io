import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import openPopup from "~/utils/helpers/popup";

interface Props {
  setInstallationId: (id?: string) => void;
  setRefreshToken: (token: number) => void;
  openPopupURL: string;
}

export default function ConnectMoreRepos({
  setInstallationId,
  setRefreshToken,
  openPopupURL,
}: Props) {
  return (
    <Button
      color="secondary"
      variant="contained"
      onClick={() => {
        openPopup({
          url: openPopupURL,
          title: "Add repository",
          width: 1000,
          onClose: () => {
            setInstallationId(undefined);
            setRefreshToken(Date.now());
          },
        });
      }}
    >
      <Box
        component="span"
        sx={{ display: { xs: "none", md: "inline-block" } }}
      >
        Connect more repositories
      </Box>
      <Box
        component="span"
        sx={{ display: { xs: "inline-block", md: "none" } }}
      >
        More repos
      </Box>
    </Button>
  );
}
