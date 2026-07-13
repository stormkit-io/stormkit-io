import React, { useContext, useEffect } from "react";
import { useNavigate } from "react-router";
import { AppContext } from "~/pages/apps/[id]/App.context";

const Deployments: React.FC = (): React.ReactElement => {
  const { app, environments } = useContext(AppContext);
  const navigate = useNavigate();
  const env = environments?.find(e => e.name === "production");

  useEffect(() => {
    navigate(`/apps/${app.id}/environments/${env?.id}/deployments`);
  }, [env]);

  return <></>;
};

export default Deployments;
