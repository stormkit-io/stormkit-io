import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import PersonAddIcon from "@mui/icons-material/PersonAdd";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import { formatDate } from "~/utils/helpers/date";
import AuthWallNewUserModal from "./AuthWallNewUserModal";
import { deleteLogins, useFetchLogins } from "./actions";

interface Props {
  app: App;
  environment: Environment;
}

export default function AuthWallUsers({ app, environment: env }: Props) {
  const [refreshToken, setRefreshToken] = useState<number>();
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [checked, setChecked] = useState<string[]>([]);
  const [showModal, setShowModal] = useState(false);
  const { loading, error, logins } = useFetchLogins({
    appId: app.id,
    envId: env.id!,
    refreshToken,
  });

  const hasAnyLogin = logins.length > 0;

  return (
    <>
      <Card
        id="auth-wall-users"
        loading={loading}
        info={
          !logins.length &&
          "There are no logins for this environment. Click the button above to add a new user."
        }
        error={error}
      >
        <CardHeader
          title="Auth users"
          subtitle="Manage users who have access to deployments in this environment. "
          actions={
            <IconButton onClick={() => setShowModal(true)}>
              <PersonAddIcon />
            </IconButton>
          }
        />
        <Box>
          {hasAnyLogin && (
            <CardRow sx={{ backgroundColor: "container.paper" }}>
              <Box
                sx={{ display: "flex", width: "100%" }}
                color="text.secondary"
              >
                <Typography sx={{ flex: 1 }}>Email</Typography>
                <Typography>Last login</Typography>
              </Box>
            </CardRow>
          )}
          {logins.map(login => (
            <CardRow
              key={login.id}
              sx={{ "&:hover": { backgroundColor: "container.paper" } }}
            >
              <Box
                sx={{ display: "flex", width: "100%", alignItems: "center" }}
              >
                <FormControlLabel
                  label={login.email}
                  sx={{ flex: 1 }}
                  control={
                    <Checkbox
                      size="small"
                      color="secondary"
                      checked={checked.indexOf(login.id) > -1}
                      onChange={() => {
                        const index = checked.indexOf(login.id);

                        if (index > -1) {
                          const clone = [...checked];
                          clone.splice(index, 1);
                          setChecked(clone);
                        } else {
                          setChecked([...checked, login.id]);
                        }
                      }}
                    />
                  }
                />
                <Typography>{formatDate(login.lastLogin)}</Typography>
              </Box>
            </CardRow>
          ))}
        </Box>
        {hasAnyLogin && (
          <CardFooter>
            <Button
              variant="contained"
              color="secondary"
              disabled={checked.length === 0}
              onClick={() => {
                setShowDeleteModal(true);
              }}
            >
              Remove selected
            </Button>
          </CardFooter>
        )}
      </Card>
      {showModal && (
        <AuthWallNewUserModal
          appId={app.id}
          envId={env.id!}
          onSuccess={() => {
            setRefreshToken(Date.now());
          }}
          onClose={() => {
            setShowModal(false);
          }}
        />
      )}
      {showDeleteModal && (
        <ConfirmModal
          onCancel={() => {
            setShowDeleteModal(false);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);

            deleteLogins({ appId: app.id, envId: env.id!, loginIds: checked })
              .then(() => {
                setRefreshToken?.(Date.now());
                setShowDeleteModal(false);
                setChecked([]);
              })
              .catch(e => {
                console.error(e);
                setError(
                  "Something went wrong while deleting logins. Check the console for more information."
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          You are about to remove{" "}
          {checked.length === 1
            ? "1 login. This user"
            : `${checked.length} logins. These users`}{" "}
          won't be able to login anymore. Please note that existing sessions
          won't be terminated until their session expiry (max 24 hours).
        </ConfirmModal>
      )}
    </>
  );
}
