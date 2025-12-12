import { useContext, useMemo } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import Box from "@mui/material/Box";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import { AppContext } from "~/pages/apps/[id]/App.context";
import MenuLink from "~/components/MenuLink";
import { envMenuItems } from "./menu_items";
import DotDotDot from "~/components/DotDotDotV2";

export default function EnvMenu() {
  const { app, environments } = useContext(AppContext);
  const { pathname } = useLocation();
  const navigate = useNavigate();

  // Deduce the envId from the pathname because we cannot access
  // the :envId url parameter, as it's included inside
  // this component as a child.
  const envId = pathname.split("/environments/")?.[1]?.split("/")?.[0];

  const env = environments.find(e => e.id === envId)!;
  const selectedEnvId = envId || "";

  const envMenu = useMemo(
    () => envMenuItems({ app, env, pathname }),
    [app, env, pathname]
  );

  if (!selectedEnvId || !env) {
    return <></>;
  }

  return (
    <Box
      bgcolor="background.paper"
      sx={{
        maxWidth: { md: "260px" },
        width: "100%",
      }}
    >
      <Box
        sx={{
          width: "100%",
          display: "flex",
          flexDirection: "column",
          pb: 2,
          mt: 2,
        }}
      >
        <Box
          sx={{
            flex: 1,
            display: "flex",
            alignItems: "center",
            mb: 2,
          }}
        >
          <Select
            variant="outlined"
            disableUnderline
            aria-label="Environment selector"
            onChange={e => {
              if (pathname.includes(`/environments/${selectedEnvId}`)) {
                navigate(
                  pathname.replace(
                    `/environments/${selectedEnvId}`,
                    `/environments/${e.target.value}`
                  )
                );
              } else {
                navigate(`/apps/${app.id}/environments/${e.target.value}`);
              }
            }}
            sx={{ border: "1px solid", borderColor: "container.border", mx: 2 }}
            fullWidth
            value={selectedEnvId || "_"}
          >
            <MenuItem value="_" disabled>
              Select an environment
            </MenuItem>
            {environments.map(e => (
              <MenuItem
                key={e.id}
                value={e.id}
                aria-label={`${e.name} environment`}
              >
                {e.env}
              </MenuItem>
            ))}
          </Select>
        </Box>
        <Box
          role="navigation"
          sx={{
            display: { xs: "none", md: "flex" },
            flexDirection: "column",
          }}
        >
          {envMenu.map(item => (
            <MenuLink
              key={item.path}
              item={item}
              sx={{
                borderBottom: "1px solid",
                borderColor: "container.border",
                mx: 2,
                px: 2,
                py: 1,
                display: "flex",
                alignItems: "center",
              }}
            />
          ))}
        </Box>
        <Box sx={{ display: { xs: "block", md: "none" }, mr: 2 }}>
          <DotDotDot items={envMenu} />
        </Box>
      </Box>
    </Box>
  );
}
