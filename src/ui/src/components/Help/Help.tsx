import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Tooltip from "@mui/material/Tooltip";
import QuestionMarkOutlined from "@mui/icons-material/HelpCenter";
import Drawer from "@mui/material/Drawer";
import { useState } from "react";
import Card from "../Card";
import CardHeader from "../CardHeader";
import { Typography } from "@mui/material";

interface Props {
  children: React.ReactNode;
  title?: string;
  subtitle?: React.ReactNode;
  buttonText?: string;
  buttonVariant?: "text" | "contained" | "outlined" | "link";
  iconOnly?: boolean;
}

export default function Help({
  children,
  title = "Help",
  subtitle,
  buttonText = "Help",
  buttonVariant = "text",
  iconOnly = false,
}: Props) {
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  return (
    <>
      {iconOnly ? (
        <Tooltip title={buttonText} arrow>
          <IconButton
            aria-label={buttonText}
            onClick={() => setIsDrawerOpen(true)}
          >
            <QuestionMarkOutlined />
          </IconButton>
        </Tooltip>
      ) : buttonVariant === "link" ? (
        <Link
          href="#"
          onClick={e => {
            e.preventDefault();
            setIsDrawerOpen(true);
          }}
        >
          {buttonText}
        </Link>
      ) : (
        <Button
          variant={buttonVariant}
          sx={{ display: "flex", alignItems: "center" }}
          onClick={() => setIsDrawerOpen(true)}
        >
          <QuestionMarkOutlined sx={{ mr: 1 }} />
          <Typography>{buttonText}</Typography>
        </Button>
      )}
      <Drawer
        anchor="right"
        open={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        sx={{ zIndex: 1600 }}
      >
        <Card
          sx={{
            minHeight: "100vh",
            minWidth: { xs: "100vw", sm: "400px" },
            maxWidth: { xs: "100vw", sm: "600px" },
          }}
        >
          <CardHeader title={title} subtitle={subtitle} />
          <Box>{children}</Box>
        </Card>
      </Drawer>
    </>
  );
}
