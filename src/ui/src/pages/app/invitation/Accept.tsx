import React, { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import Error404 from "~/components/Errors/Error404";
import { useParseAppId } from "./actions";

const InvitationAccept: React.FC = (): React.ReactElement => {
  const appId = useParseAppId();
  const navigate = useNavigate();

  useEffect(() => {
    if (appId) {
      navigate(`/apps/${appId}/team`);
    }
  }, [appId]);

  if (appId) {
    return <></>;
  }

  return (
    <Error404>
      The invitation token is invalid. Make sure that the username and provider
      match your login credentials.
    </Error404>
  );
};

export default InvitationAccept;
