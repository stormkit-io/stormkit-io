import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import EmptyPage from "~/components/EmptyPage";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import { useFetchDeployments } from "./actions";
import Deployment from "~/shared/deployments/DeploymentRow";

export default function Deployments() {
  const { teams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams });
  const [refreshToken, setRefreshToken] = useState(0);
  const { deployments, loading, error } = useFetchDeployments({
    from: 0,
    refreshToken,
    filters: { teamId: team?.id },
  });

  return (
    <Box maxWidth="lg" sx={{ width: "100%" }}>
      <Card loading={loading} error={error} contentPadding={false}>
        <CardHeader
          title="Team Deployments"
          subtitle="Display all of your team's deployments in a single view"
        />
        <Box>
          {deployments.map(deployment => (
            <Deployment
              key={deployment.id}
              deployment={deployment}
              setRefreshToken={setRefreshToken}
              showProject
            />
          ))}
        </Box>
        {!loading && !error && !deployments.length && (
          <EmptyPage>
            It's quite empty in here. <br />
            Go back to your{" "}
            <Link href="/" sx={{ fontWeight: "bold" }}>
              Apps
            </Link>{" "}
            to start deploying your website.
          </EmptyPage>
        )}
      </Card>
    </Box>
  );
}
