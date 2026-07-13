import { useContext, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import Spinner from "~/components/Spinner";
import { useInvitationAccept } from "./actions";
import { AuthContext } from "~/pages/auth/Auth.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";

export default function InvitationAccept() {
  const navigate = useNavigate();
  const { reloadTeams } = useContext(AuthContext);
  const [params] = useSearchParams();
  const { team, error, loading } = useInvitationAccept({
    token: params.get("token") || "",
  });

  useEffect(() => {
    if (team) {
      reloadTeams?.();

      // Navigate to team ID because we don't know if slugs are unique or not.
      navigate(`/${team.id}`, { replace: true });
    }
  }, [team]);

  if (loading) {
    return <Spinner pageCenter />;
  }

  return (
    <Card error={error}>
      <CardHeader title="Enrollment failed" />
    </Card>
  );
}
