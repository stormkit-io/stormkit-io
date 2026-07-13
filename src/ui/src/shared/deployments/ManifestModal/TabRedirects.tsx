import { useContext } from "react";
import Box from "@mui/material/Box";
import Table from "@mui/material/Table";
import TableRow from "@mui/material/TableRow";
import TableHead from "@mui/material/TableHead";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import Alert from "@mui/material/Alert";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import EmptyPage from "~/components/EmptyPage";

interface Props {
  redirects?: Redirects[] | null;
}

interface RedirectStatusProps {
  status?: number;
  isAbsolute: boolean;
}

function redirectStatus({ status, isAbsolute }: RedirectStatusProps): string {
  if (status) {
    const mod = status % 300;

    // Is redirect between 300 and 307?
    if (mod >= 0 && mod < 7) {
      return `Redirect ${status}`;
    }
  }

  return isAbsolute ? "Proxy" : "Path rewrite";
}

export default function TabRedirects({ redirects }: Props) {
  const { environment } = useContext(EnvironmentContext);

  if (!redirects && !environment.build?.redirects?.length) {
    return (
      <EmptyPage sx={{ my: 6 }}>
        <>
          <Typography component="span" sx={{ mb: 4, display: "block" }}>
            This deployment has no redirects.
            <br />
            Create a top-level{" "}
            <span className="font-mono text-xs text-white font-bold">
              redirects.json
            </span>{" "}
            to add server side redirects.{" "}
          </Typography>
          <Button
            href="https://www.stormkit.io/docs/features/redirects-and-path-rewrites"
            variant="contained"
            color="secondary"
          >
            Learn more
          </Button>
        </>
      </EmptyPage>
    );
  }

  return (
    <Box>
      {environment.build.redirects?.length && (
        <Alert color="info" sx={{ mb: 4 }}>
          <Typography>
            Environment level redirects active. They take precedence over
            deployment redirects.
          </Typography>
        </Alert>
      )}
      <Table>
        <TableHead>
          <TableRow
            sx={{
              bgcolor: "container.paper",
              fontWeight: "bold",
            }}
          >
            <TableCell>From</TableCell>
            <TableCell>To</TableCell>
            <TableCell>Status</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {redirects?.map(redirect => (
            <TableRow key={redirect.from}>
              <TableCell>{redirect.from}</TableCell>
              <TableCell>{redirect.to}</TableCell>
              <TableCell>
                {redirectStatus({
                  status: redirect.status,
                  isAbsolute: redirect.to?.startsWith("http"),
                })}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );
}
