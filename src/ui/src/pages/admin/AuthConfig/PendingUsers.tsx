import { useState } from "react";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import Table from "@mui/material/Table";
import Checkbox from "@mui/material/Checkbox";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import Button from "@mui/material/Button";
import TableRow from "@mui/material/TableRow";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import api from "~/utils/api/Api";
import { formatDate } from "~/utils/helpers/date";
import { useFetchPendingUsers } from "./actions";

export default function PendingUsers() {
  const [refreshToken, setRefreshToken] = useState(0);
  const { loading, error, users } = useFetchPendingUsers({ refreshToken });
  const [checked, setChecked] = useState<Record<string, boolean>>({});
  const [selectAll, setSelectAll] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState<"approve" | "reject">();

  const hasPendingUsers = !loading && !error && users.length > 0;

  return (
    <Card
      loading={loading}
      error={error}
      sx={{ backgroundColor: "container.transparent", mt: 4 }}
      info={!hasPendingUsers && "There are no pending users at the moment."}
      contentPadding={false}
    >
      <CardHeader
        title="Pending Users"
        subtitle="List of users awaiting approval"
      />
      <Box sx={{ mx: 4 }}>
        {users.length > 0 && (
          <Table>
            <TableHead>
              <TableRow>
                <TableCell sx={{ width: 48 }}>
                  <Checkbox
                    size="small"
                    checked={selectAll}
                    onChange={e => {
                      setSelectAll(e.target.checked);
                      setChecked(
                        users.reduce((acc, user) => {
                          acc[user.id] = e.target.checked;
                          return acc;
                        }, {} as Record<string, boolean>)
                      );
                    }}
                  />
                </TableCell>
                <TableCell>
                  <Typography component="span" color="text.secondary">
                    Email
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography component="span" color="text.secondary">
                    Display name
                  </Typography>
                </TableCell>
                <TableCell sx={{ textAlign: "right" }}>
                  <Typography component="span" color="text.secondary">
                    Date requested
                  </Typography>
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users?.map(user => (
                <TableRow key={user.id} title={user.email}>
                  <TableCell>
                    <Checkbox
                      size="small"
                      checked={checked[user.id] || false}
                      onChange={() => {
                        setSelectAll(false);
                        setChecked(prev => ({
                          ...prev,
                          [user.id]: !prev[user.id],
                        }));
                      }}
                    />
                  </TableCell>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>{user.displayName}</TableCell>
                  <TableCell sx={{ textAlign: "right" }}>
                    {formatDate(user.memberSince)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        <CardFooter
          sx={{
            justifyContent: "flex-end",
            mt: 4,
            display: !hasPendingUsers ? "none" : "flex",
          }}
        >
          <Button
            variant="contained"
            color="primary"
            sx={{ mr: 2 }}
            disabled={
              Object.values(checked).filter(value => value).length === 0
            }
            onClick={() => {
              setConfirmOpen("reject");
            }}
          >
            Reject selected
          </Button>
          <Button
            variant="contained"
            color="secondary"
            disabled={
              Object.values(checked).filter(value => value).length === 0
            }
            onClick={() => {
              setConfirmOpen("approve");
            }}
          >
            Approve selected
          </Button>
        </CardFooter>
      </Box>
      {confirmOpen && (
        <ConfirmModal
          title={confirmOpen === "approve" ? "Approve Users" : "Reject Users"}
          onCancel={() => {
            setConfirmOpen(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            api
              .post("/admin/users/manage", {
                userIds: Object.keys(checked).filter(id => checked[id]),
                action: confirmOpen,
              })
              .then(() => {
                setRefreshToken(Date.now());
                setChecked({});
                setSelectAll(false);
                setConfirmOpen(undefined);
              })
              .catch(() => {
                setError(
                  "Something went wrong while managing the selected users. Please try again later."
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          You are about to {confirmOpen} the selected users.
        </ConfirmModal>
      )}
    </Card>
  );
}
