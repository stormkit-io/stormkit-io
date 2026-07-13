import { useContext, useEffect } from "react";
import Box from "@mui/material/Box";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { AuthContext } from "~/pages/auth/Auth.context";
import Spinner from "~/components/Spinner";
import FormAppSettings from "./_components/FormAppSettings";
import FormTriggerDeploys from "./_components/FormTriggerDeploys";
import FormOutboundWebhooks from "./_components/FormOutboundWebhooks";
import FormDangerZone from "./_components/FormDangerZone";
import FormMigrateApp from "./_components/FormMigrateApp";
import * as actions from "./actions";

const { useFetchAdditionalSettings } = actions;

export default function Settings() {
  const { app, environments } = useContext(AppContext);
  const { teams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams, app });
  const { settings, loading, setSettings } = useFetchAdditionalSettings({
    app,
  });

  const hasWriteAccess =
    team?.currentUserRole === "owner" || team?.currentUserRole === "admin";

  useEffect(() => {
    if (window.location.hash) {
      setTimeout(() => {
        document
          .querySelector(window.location.hash)
          ?.scrollIntoView({ behavior: "smooth" });
      }, 500);
    }
  }, []);

  if (loading) {
    return (
      <Box
        sx={{
          width: "100%",
          display: "flex",
          justifyContent: "center",
          m: "auto",
        }}
        maxWidth="md"
      >
        <Spinner secondary />
      </Box>
    );
  }

  return (
    <Box sx={{ m: "auto" }} maxWidth="md">
      <FormAppSettings app={app} additionalSettings={settings} />

      <FormTriggerDeploys
        app={app}
        additionalSettings={settings}
        setAdditionalSettings={setSettings}
        environments={environments}
      />

      <FormOutboundWebhooks app={app} />

      {hasWriteAccess && <FormMigrateApp teams={teams!} app={app} />}

      {hasWriteAccess && <FormDangerZone app={app} />}
    </Box>
  );
}
