import { useContext, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import RollbackIcon from "@mui/icons-material/RefreshOutlined";
import ShieldIcon from "@mui/icons-material/Shield";
import CloudIcon from "@mui/icons-material/Cloud";
import Skeleton from "@mui/material/Skeleton";
import qs from "query-string";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Logo from "~/components/Logo";
import { AuthContext } from "./Auth.context";
import BasicAuthRegister from "./BasicAuthRegister";
import BasicAuthLogin from "./BasicAuthLogin";
import ProviderAuth from "./ProviderAuth";
import { Link } from "@mui/material";

export default function Auth() {
  const { user, providers } = useContext(AuthContext);
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    if (user) {
      const { redirect = "/", template } = qs.parse(
        location.search.replace("?", "")
      );

      if (typeof redirect === "string") {
        if (template !== undefined) {
          navigate(`/clone?template=${template}`);
        } else {
          navigate(redirect);
        }
      }
    }
  }, [user]);

  if (user) {
    return null;
  }

  const isBasicAuth =
    !providers?.github && !providers?.gitlab && !providers?.bitbucket;

  const loading = false;

  return (
    <Box>
      <Box
        maxWidth="lg"
        sx={{
          left: { md: "50%" },
          transform: { md: "translateX(-50%)" },
          position: { md: "fixed" },
          pl: { xs: 3, md: 0 },
          pt: { xs: 2, md: 4 },
          mb: { xs: 4, md: 0 },
          width: "100%",
        }}
      >
        <Logo iconSize={150} />
      </Box>
      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" },
          minHeight: { md: "100vh" },
        }}
      >
        <Box
          sx={{
            maxWidth: "600px",
            justifySelf: { md: "flex-end" },
            alignSelf: "center",
            p: 4,
            pt: 5.5, // to offset logo
            mb: { xs: 8, md: 0 },
          }}
        >
          <Typography sx={{ lineHeight: 2, fontSize: 14 }}>
            /def/{" "}
            <Typography component="span" color="secondary.dark">
              Noun.
            </Typography>
            <br />
            1. Serverless app development platform.
            <br />
            2. A set of tools built to save dev-ops time for your Javascript
            application.
          </Typography>
          <Box component="ul" sx={{ mt: 4 }}>
            <Typography
              component="li"
              sx={{ mb: 2, fontSize: 14 }}
              color="text.secondary"
            >
              <RollbackIcon sx={{ mr: 1 }} /> Environments with instant
              rollbacks
            </Typography>
            <Typography
              component="li"
              sx={{ mb: 2, fontSize: 14 }}
              color="text.secondary"
            >
              <ShieldIcon sx={{ mr: 1 }} /> Custom domains &amp; automated TLS
            </Typography>
            <Typography
              component="li"
              sx={{ fontSize: 14 }}
              color="text.secondary"
            >
              <CloudIcon sx={{ mr: 1 }} /> Serverless functions
            </Typography>
          </Box>

          <Typography
            sx={{
              mt: 4,
              pt: 2,
              textAlign: "center",
              borderTop: "1px solid",
              borderColor: "rgba(255, 255, 255, 0.1)",
            }}
            color="text.secondary"
          >
            By using Stormkit, you're agreeing to our{" "}
            <Link
              href="https://www.stormkit.io/policies/terms"
              target="_blank"
              rel="noopener noreferrer"
            >
              terms and services
            </Link>
            .
          </Typography>
        </Box>
        <Box
          sx={{
            minHeight: { md: "100vh" },
            display: "flex",
            alignSelf: "center",
            bgcolor: "background.paper",
          }}
        >
          <Box sx={{ maxWidth: "600px", width: "100%", alignSelf: "center" }}>
            {loading ? (
              <Box sx={{ px: 4 }}>
                <Skeleton variant="rectangular" height={30} sx={{ mb: 4 }} />
                <Skeleton variant="rectangular" height={30} sx={{ mb: 4 }} />
                <Skeleton variant="rectangular" height={30} />
              </Box>
            ) : !isBasicAuth ? (
              <ProviderAuth providers={providers} />
            ) : providers?.basicAuth === "enabled" ? (
              <BasicAuthLogin />
            ) : (
              <BasicAuthRegister />
            )}
          </Box>
        </Box>
      </Box>
    </Box>
  );
}
