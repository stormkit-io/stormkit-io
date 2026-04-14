import { useContext } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { formatDate } from "~/utils/helpers/date";
import { AuthUser, useFetchAuthUsers } from "./actions";

function displayName(user: AuthUser): string {
  const name = [user.firstName, user.lastName].filter(Boolean).join(" ");
  return name || "-";
}

export default function AuthUsers() {
  const { environment: env } = useContext(EnvironmentContext);
  const { loading, error, users, hasNextPage, loadMore } = useFetchAuthUsers({
    envId: env.id!,
  });

  const hasUsers = !loading && !error && users.length > 0;

  return (
    <Card
      loading={loading}
      error={error}
      sx={{ width: "100%" }}
      info={!hasUsers && !loading && "No registered users yet."}
      contentPadding={false}
    >
      <CardHeader
        title="Registered Users"
        subtitle="Users registered through the authentication providers"
      />
      <Box sx={{ mx: 4 }}>
        {hasUsers && (
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>
                  <Typography component="span" color="text.secondary">
                    Email
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography component="span" color="text.secondary">
                    Name
                  </Typography>
                </TableCell>
                <TableCell sx={{ textAlign: "right" }}>
                  <Typography component="span" color="text.secondary">
                    Registered
                  </Typography>
                </TableCell>
                <TableCell sx={{ textAlign: "right" }}>
                  <Typography component="span" color="text.secondary">
                    Last login
                  </Typography>
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map(user => (
                <TableRow key={user.id}>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>{displayName(user)}</TableCell>
                  <TableCell sx={{ textAlign: "right" }}>
                    {formatDate(user.createdAt)}
                  </TableCell>
                  <TableCell sx={{ textAlign: "right" }}>
                    {user.lastLoginAt ? formatDate(user.lastLoginAt) : "-"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        {hasNextPage && (
          <CardFooter sx={{ justifyContent: "center", mt: 2 }}>
            <Button variant="outlined" onClick={loadMore} disabled={loading}>
              Load more
            </Button>
          </CardFooter>
        )}
      </Box>
    </Card>
  );
}
