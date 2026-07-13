import { useContext } from "react";
import { RootContext } from "~/pages/Root.context";
import EmptyPage from "~/components/EmptyPage";
import AuthWallConfig from "./AuthWallConfig";
import AuthWallUsers from "./AuthWallUsers";

interface Props {
  app: App;
  environment: Environment;
}

export default function TabAuthWall({ app, environment: env }: Props) {
  const { details } = useContext(RootContext);

  if (details?.license?.edition === "community") {
    return <EmptyPage paymentRequired />;
  }

  return (
    <>
      <AuthWallConfig app={app} environment={env} />
      <AuthWallUsers app={app} environment={env} />
    </>
  );
}
