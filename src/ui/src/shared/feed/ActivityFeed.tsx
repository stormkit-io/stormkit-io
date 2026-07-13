import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import { AuthContext } from "~/pages/auth/Auth.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import EmptyPage from "~/components/EmptyPage";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import AuditMessage from "./AuditMessage";
import { useFetchAudits } from "./actions";

export default function Audit() {
  const { teams } = useContext(AuthContext);
  const { app } = useContext(AppContext);
  const selectedTeam = useSelectedTeam({ teams });
  const [nextPage, loadNextPage] = useState(0);

  const { audits, loading, error, hasNextPage, paymentRequired } =
    useFetchAudits({
      teamId: selectedTeam!.id,
      appId: app?.id,
      nextPage,
    });

  return (
    <Box maxWidth="lg" sx={{ width: "100%" }}>
      <Card loading={loading} error={error} contentPadding={false}>
        <CardHeader
          title="Activity Feed"
          subtitle="Track actions made by members of your team"
          sx={{ pb: 2 }}
        />
        <Box>
          {audits.map(audit => (
            <AuditMessage audit={audit} key={audit.id} />
          ))}
        </Box>
        {!loading && !error && !audits.length && (
          <EmptyPage paymentRequired={paymentRequired}>
            Looks like nothing in here. <br />
            All actions made by your team members will be tracked here.
          </EmptyPage>
        )}
        {hasNextPage && (
          <CardFooter sx={{ textAlign: "center" }}>
            <Button
              variant="text"
              loading={loading}
              onClick={() => {
                loadNextPage(Date.now());
              }}
            >
              Load more
            </Button>
          </CardFooter>
        )}
      </Card>
    </Box>
  );
}
