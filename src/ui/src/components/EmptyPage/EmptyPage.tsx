import type { SxProps } from "@mui/material";
import React from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import emptyListSvg from "~/assets/images/empty-list.svg";
import UpgradeButton from "~/components/UpgradeButton";

interface Props {
  sx?: SxProps;
  children?: React.ReactNode;
  paymentRequired?: boolean;
}

export default function EmptyList({ sx, children, paymentRequired }: Props) {
  return (
    <Box sx={{ textAlign: "center", my: 12, ...sx }}>
      <img src={emptyListSvg} alt="Empty app list" className="m-auto" />
      {paymentRequired ? (
        <>
          <Typography
            fontSize="medium"
            sx={{ mt: 6, mb: 2, fontWeight: "bold" }}
          >
            You need to upgrade your plan to access this feature.
          </Typography>
          <UpgradeButton fullWidth={false} />
        </>
      ) : (
        <Typography fontSize="medium" sx={{ mt: 6 }}>
          {children || <>It's quite empty in here.</>}
        </Typography>
      )}
    </Box>
  );
}
