import { useContext } from "react";
import { useNavigate, useMatch } from "react-router-dom";
import Box from "@mui/material/Box";
import { AuthContext } from "~/pages/auth/Auth.context";
import Error404 from "~/components/Errors/Error404";
import UserProfile from "./_components/UserProfile";
import ConnectedAccounts from "./_components/ConnectedAccounts";
import APIKeys from "./_components/APIKeys";
import ShareSessionModal from "./_components/ShareSessionModal";

export default function Account() {
  const { user, accounts, metrics } = useContext(AuthContext);
  const navigate = useNavigate();
  const isShareRoute = useMatch("/user/account/share");

  if (!user) {
    return <Error404 />;
  }

  return (
    <Box sx={{ mx: "auto", mt: 2 }} maxWidth="lg">
      <UserProfile user={user} metrics={metrics} />
      <ConnectedAccounts accounts={accounts!} />
      <APIKeys user={user} />
      {isShareRoute && (
        <ShareSessionModal onClose={() => navigate("/user/account")} />
      )}
    </Box>
  );
}
