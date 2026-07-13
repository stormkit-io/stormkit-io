import { useEffect, useState } from "react";
import CircularProgress from "@mui/material/CircularProgress";
import SearchIcon from "@mui/icons-material/Search";
import TrashIcon from "@mui/icons-material/Delete";
import PersonSearchIcon from "@mui/icons-material/PersonSearch";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Table from "@mui/material/Table";
import Head from "@mui/material/TableHead";
import Body from "@mui/material/TableBody";
import Tr from "@mui/material/TableRow";
import Td from "@mui/material/TableCell";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import ConfirmModal from "~/components/ConfirmModal";
import DotDotDot from "~/components/DotDotDotV2";
import api from "~/utils/api/Api";
import { formatDate } from "~/utils/helpers/date";

const useFetchCloudApp = (url: string) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [app, setApp] = useState<App>();
  const [user, setUser] = useState<User>();

  useEffect(() => {
    setLoading(false);
    setError(undefined);

    if (!url) {
      return;
    }

    setLoading(true);

    api
      .fetch<{ app: App; user: User }>(
        "/admin/cloud/app?url=" + encodeURIComponent(url)
      )
      .then(({ app, user }) => {
        setApp(app);
        setUser(user);
      })
      .catch(() => {
        setError("Something went wrong while fetching app");
        setApp(undefined);
        setUser(undefined);
      })
      .finally(() => setLoading(false));
  }, [url]);

  return { loading, error, app, user };
};

/**
 * This page is used to manage malicious users.
 * See https://www.stormkit.io/policies/terms for acceptable usage.
 */
export default function CloudApps() {
  const [search, setSearch] = useState("");
  const [success, setSuccess] = useState<string>();
  const { error, loading, app, user } = useFetchCloudApp(search);
  const [appToBeDeleted, setAppToBeDeleted] = useState<App>();

  return (
    <Card
      error={error}
      success={success}
      sx={{ backgroundColor: "container.transparent" }}
      info={
        search &&
        !app &&
        !loading &&
        "There is no application based on your search criteria."
      }
      contentPadding={false}
    >
      <CardHeader
        title="Apps"
        subtitle="Use the input below to search for apps and manage them."
      />
      <Box sx={{ px: 4, mb: 4 }}>
        <TextField
          variant="filled"
          label="Search"
          placeholder="Search an app by it's display or domain name and press Enter"
          onKeyUp={e => {
            if (e.key !== "Enter") {
              return;
            }

            setSearch((e.target as HTMLInputElement).value);
          }}
          slotProps={{
            inputLabel: {
              shrink: true,
            },
            input: {
              endAdornment: loading ? (
                <CircularProgress size={16} />
              ) : (
                <SearchIcon />
              ),
            },
          }}
          fullWidth
        />
      </Box>
      {app && (
        <Box sx={{ px: 4 }}>
          <Table>
            <Head>
              <Tr>
                <Td>ID</Td>
                <Td>Display name</Td>
                <Td>Created at</Td>
                <Td>User id</Td>
                <Td>User display</Td>
                <Td>Email</Td>
                <Td></Td>
              </Tr>
            </Head>
            <Body>
              <Tr>
                <Td>{app.id}</Td>
                <Td>{app.displayName}</Td>
                <Td>{formatDate(app.createdAt)}</Td>
                <Td>{user?.id}</Td>
                <Td>{user?.displayName}</Td>
                <Td>{user?.email}</Td>
                <Td sx={{ textAlign: "right", pr: 0 }}>
                  <DotDotDot
                    label="expand"
                    items={[
                      {
                        text: "Delete",
                        icon: <TrashIcon />,
                        onClick: () => setAppToBeDeleted(app),
                      },
                      {
                        // This is to impersonate the user and provide them support.
                        // This implementation is temporary, as it provides opportunity for
                        // abuse. A more robust implementation will be provided in the future,
                        // where the user creates a token and passes that token to support.
                        text: "Impersonate",
                        icon: <PersonSearchIcon />,
                        onClick: () => {
                          api
                            .post<{ token: string }>(
                              "/admin/cloud/impersonate",
                              {
                                userId: user?.id,
                              }
                            )
                            .then(({ token }) => {
                              if (token) {
                                api.setAuthToken(token);
                                window.location.assign("/");
                              }
                            });
                        },
                      },
                    ]}
                  />
                </Td>
              </Tr>
            </Body>
          </Table>
        </Box>
      )}

      {appToBeDeleted && (
        <ConfirmModal
          onCancel={() => {
            setAppToBeDeleted(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);
            setSuccess(undefined);

            api
              .delete("/admin/cloud/app?appId=" + appToBeDeleted.id)
              .then(() => {
                setSuccess("The app has been deleted successfully.");
                setAppToBeDeleted(undefined);
              })
              .catch(() => {
                setError(
                  "Something went wrong while deleting the app and user."
                );
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          This will delete the application and soft delete the user.
        </ConfirmModal>
      )}
      <CardFooter />
    </Card>
  );
}
