import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { green, red } from "@mui/material/colors";
import PublicIcon from "@mui/icons-material/Public";
import PublicOffIcon from "@mui/icons-material/PublicOff";
import Spinner from "~/components/Spinner";

interface StatusProps {
  status?: number;
  loading?: boolean;
}

export default function DomainStatus({ status, loading }: StatusProps) {
  const isSuccess = status === 200;

  return (
    <Box sx={{ display: "flex", alignItems: "center" }}>
      {loading && <Spinner width={4} height={4} />}
      {!loading && (
        <Typography
          component="span"
          sx={{
            display: "flex",
            alignItems: "center",
            color: isSuccess ? green[500] : red[500],
          }}
        >
          {isSuccess && <PublicIcon sx={{ fontSize: 15, mr: 1 }} />}
          {!isSuccess && <PublicOffIcon sx={{ fontSize: 15, mr: 1 }} />}
          <Typography component="span" color={isSuccess ? "green" : "red"}>
            {status}
          </Typography>
        </Typography>
      )}
    </Box>
  );
}
