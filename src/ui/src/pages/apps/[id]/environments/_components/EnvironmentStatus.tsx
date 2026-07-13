import Typography from "@mui/material/Typography";
import { green, red } from "@mui/material/colors";
import { useFetchStatus } from "../actions";

interface Props {
  env: Environment;
  app: App;
}

const isSuccessStatus = (status: number | undefined): boolean => {
  if (!status) {
    return false;
  }

  return status % 200 < 10 || status % 300 < 10;
};

export default function EnvironmentStatus({ env, app }: Props) {
  const endpoint = env?.preview || "";

  const { status } = useFetchStatus({
    environment: env,
    app,
    domain: endpoint,
  });

  if (!env.published) {
    return;
  }

  return (
    <Typography
      component="span"
      sx={{ color: isSuccessStatus(status) ? green[500] : red[500] }}
    >
      {status}
    </Typography>
  );
}
