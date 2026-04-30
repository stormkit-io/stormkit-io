import { useContext, useState } from "react";
import LockIcon from "@mui/icons-material/Lock";
import LaunchIcon from "@mui/icons-material/Launch";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import { subscriptionLink } from "~/utils/billing";
import Modal from "~/components/Modal";

interface Props {
  fullWidth?: boolean;
  text?: string;
  variant?: "contained" | "outlined" | "text";
}

export default function UpgradeButton({
  fullWidth = true,
  text = "Upgrade to enterprise",
  variant = "contained",
}: Props) {
  const { user } = useContext(AuthContext);
  const { details } = useContext(RootContext);
  const [infoOpen, setInfoOpen] = useState(false);
  const Icon = text === "Upgrade to enterprise" ? LockIcon : LaunchIcon;
  const href = subscriptionLink(
    user?.package.id,
    user?.email,
    details?.stormkit?.edition,
  );

  const showInfoModal =
    details?.stormkit?.edition !== "cloud" && !user?.isAdmin;

  const externalLinkProps = showInfoModal
    ? {
        onClick: (e: React.MouseEvent) => {
          e.preventDefault();
          setInfoOpen(true);
        },
      }
    : {
        href,
        target: "_blank" as const,
        rel: "noopener noreferrer",
      };

  return (
    <>
      {variant === "text" ? (
        <Link
          color="secondary"
          sx={{ textDecoration: "underline !important", cursor: "pointer" }}
          {...externalLinkProps}
        >
          {text}
        </Link>
      ) : (
        <Button
          variant={variant}
          color="secondary"
          fullWidth={fullWidth}
          sx={{ mb: 1 }}
          startIcon={<Icon sx={{ fontSize: 16 }} />}
          {...externalLinkProps}
        >
          {text}
        </Button>
      )}
      {showInfoModal && (
        <Modal
          open={infoOpen}
          onClose={() => setInfoOpen(false)}
          maxWidth="sm"
          title="License Management"
        >
          <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 2 }}>
            <Typography variant="h2">
              <InfoOutlinedIcon sx={{ mr: 1 }} />
              License management
            </Typography>
            <Typography color="textSecondary">
              In self-hosted environments, the license is managed by admins.
            </Typography>
            <Typography color="textSecondary">
              Contact your administrator to upgrade your subscription or to get
              more information about your current plan.
            </Typography>
            <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
              <Button
                color="secondary"
                variant="contained"
                onClick={() => setInfoOpen(false)}
              >
                Close
              </Button>
            </Box>
          </Box>
        </Modal>
      )}
    </>
  );
}
