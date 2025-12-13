import React, { useContext } from "react";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import InfoIcon from "@mui/icons-material/Info";
import Card from "~/components/Card";
import DeployButton from "~/layouts/AppLayout/_components/DeployButton";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { useFetchStatus, isEmpty } from "../../actions";
import DomainStatus from "./DomainStatus";

interface ColumnProps {
  children: React.ReactNode;
}

function Column({ children }: ColumnProps) {
  return (
    <Box
      sx={{
        pr: 2,
        mr: 2,
        borderRight: "1px solid",
        borderColor: "container.transparent",
      }}
    >
      {children}
    </Box>
  );
}

export default function EnvironmentHeader() {
  const { environments } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const { app } = useContext(AppContext);
  const domainName = environment.preview;

  const { status, loading } = useFetchStatus({
    environment,
    domain: domainName,
    app,
  });

  return (
    <Card sx={{ width: "100%", mb: 2 }}>
      <Box
        fontSize="small"
        sx={{
          width: "100%",
          display: "flex",
          alignItems: "center",
        }}
      >
        <Column>
          <DomainStatus loading={loading} status={status} />
        </Column>
        <Column>
          <Typography
            component="a"
            href={domainName}
            target="_blank"
            rel="noreferrer noopener"
            sx={{ display: { xs: "none", md: "inline" } }}
          >
            {domainName.replace(/^https?:\/\//, "")}
          </Typography>
        </Column>
        <Column>
          <span className="fa fa-code-branch mr-2" />
          <Typography component="span">{environment.branch}</Typography>
        </Column>
        <Box sx={{ flex: 1, display: { xs: "none", md: "inherit" } }}>
          {!environment.lastDeploy?.id ? (
            <>
              <Tooltip
                title={"Deploy your app to start serving your application."}
                placement="bottom"
                arrow
              >
                <Typography component="span">
                  <InfoIcon sx={{ mr: 1 }} />
                  Not yet deployed
                </Typography>
              </Tooltip>
            </>
          ) : !environment.published ? (
            <>
              <Tooltip
                title={"Publish a deployment to serve your app."}
                placement="bottom"
                arrow
              >
                <Typography component="span">
                  <InfoIcon sx={{ mr: 1 }} />
                  Not yet published
                </Typography>
              </Tooltip>
            </>
          ) : (
            <Box
              sx={{
                mr: 2,
                mb: { xs: 2, md: 0 },
                flex: 1,
              }}
            >
              <Typography component="span">Published</Typography>
              {!isEmpty(environment.published) &&
                Array.isArray(environment.published) && (
                  <>
                    :{" "}
                    {environment.published.length > 1 ? (
                      "multiple deployments"
                    ) : (
                      <Link
                        key={environment.published[0].deploymentId}
                        href={`/apps/${environment.appId}/environments/${environment.id}/deployments/${environment.published[0].deploymentId}`}
                      >
                        {environment.published[0].deploymentId}
                      </Link>
                    )}
                    {status?.toString()[0] !== "2" && (
                      <Tooltip
                        title={
                          <Box sx={{ p: 2 }}>
                            <Typography className="text-center">
                              The endpoint returns 404.
                              <br />
                              Click on{" "}
                              {environment.published?.length > 1
                                ? "a"
                                : "the"}{" "}
                              deployment to debug.
                            </Typography>
                            {environment.published?.map(p => (
                              <Box
                                sx={{
                                  textAlign: "center",
                                  mt: 2,
                                }}
                                key={p.deploymentId}
                              >
                                <Button variant="contained" color="secondary">
                                  #{p.deploymentId}
                                </Button>
                              </Box>
                            ))}
                          </Box>
                        }
                      >
                        <Box
                          component="span"
                          sx={{ ml: 1 }}
                          className="fas fa-triangle-exclamation"
                          aria-label="Deployment not found"
                        />
                      </Tooltip>
                    )}
                  </>
                )}
            </Box>
          )}
        </Box>
        <Box sx={{ textAlign: "right", flex: 1 }}>
          <DeployButton
            app={app}
            environments={environments}
            selectedEnvId={environment.id!}
          />
        </Box>
      </Box>
    </Card>
  );
}
