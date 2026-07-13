import React, { useContext, useEffect } from "react";
import { useParams, useNavigate } from "react-router";
import { AppContext } from "~/pages/apps/[id]/App.context";
import Spinner from "~/components/Spinner";
import Container from "~/components/Container";
import { useFetchDeployment } from "~/pages/apps/[id]/environments/[env-id]/deployments/actions";

const Deployment: React.FC = () => {
  const navigate = useNavigate();
  const { deploymentId } = useParams();
  const { app } = useContext(AppContext);
  const { deployment, loading } = useFetchDeployment({
    deploymentId,
  });

  const envId = deployment?.envId;
  const { id } = deployment || {};

  useEffect(() => {
    if (deployment) {
      navigate(`/apps/${app.id}/environments/${envId}/deployments/${id}`);
    }
  }, [envId, id]);

  if (loading) {
    return (
      <Container>
        <div className="flex w-full justify-center">
          <Spinner />
        </div>
      </Container>
    );
  }

  return <></>;
};

export default Deployment;
