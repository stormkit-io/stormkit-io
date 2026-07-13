import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { formatDate } from "~/utils/helpers/date";
import { AuthUser, deleteAuthUser, useFetchAuthUsers } from "./actions";
import EditUserModal from "./EditUserModal";

function displayName(user: AuthUser): string {
  const name = [user.firstName, user.lastName].filter(Boolean).join(" ");
  return name || "-";
}

export default function AuthUsers() {
  const { environment: env } = useContext(EnvironmentContext);
  const [refreshToken, setRefreshToken] = useState<number>();
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement>();
  const [selected, setSelected] = useState<AuthUser>();
  const [editUser, setEditUser] = useState<AuthUser>();
  const [deleteUser, setDeleteUser] = useState<AuthUser>();
  const { loading, error, users, hasNextPage, loadMore } = useFetchAuthUsers({
    envId: env.id!,
    refreshToken,
  });

  const hasUsers = !loading && !error && users.length > 0;

  const closeMenu = () => {
    setMenuAnchor(undefined);
    setSelected(undefined);
  };

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
                <TableCell sx={{ width: 48 }} />
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
                  <TableCell sx={{ textAlign: "right", py: 0 }}>
                    <IconButton
                      size="small"
                      aria-label="User actions"
                      onClick={e => {
                        setMenuAnchor(e.currentTarget);
                        setSelected(user);
                      }}
                    >
                      <MoreVertIcon fontSize="small" />
                    </IconButton>
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
      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={closeMenu}
      >
        <MenuItem
          onClick={() => {
            setEditUser(selected);
            closeMenu();
          }}
        >
          <ListItemIcon>
            <EditIcon fontSize="small" />
          </ListItemIcon>
          Edit
        </MenuItem>
        <MenuItem
          onClick={() => {
            setDeleteUser(selected);
            closeMenu();
          }}
        >
          <ListItemIcon>
            <DeleteIcon fontSize="small" />
          </ListItemIcon>
          Delete
        </MenuItem>
      </Menu>
      {editUser && (
        <EditUserModal
          envId={env.id!}
          user={editUser}
          onClose={() => setEditUser(undefined)}
          onSuccess={() => setRefreshToken(Date.now())}
        />
      )}
      {deleteUser && (
        <ConfirmModal
          onCancel={() => setDeleteUser(undefined)}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);

            deleteAuthUser({ envId: env.id!, userId: deleteUser.id })
              .then(() => {
                setRefreshToken(Date.now());
                setDeleteUser(undefined);
              })
              .catch(e => {
                console.error(e);
                setError(
                  "Something went wrong while deleting the user. Check the console for more information.",
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          You are about to delete{" "}
          <Box component="span" sx={{ fontWeight: "bold" }}>
            {deleteUser.email}
          </Box>
          . This removes the user and all of their linked provider logins. This
          action cannot be undone.
        </ConfirmModal>
      )}
    </Card>
  );
}
