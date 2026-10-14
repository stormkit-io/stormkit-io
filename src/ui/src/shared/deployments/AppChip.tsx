import type { SxProps } from "@mui/material";
import React from "react";
import Chip from "@mui/material/Chip";
import Avatar from "@mui/material/Avatar";
import BoltIcon from "@mui/icons-material/Bolt";
import Dot from "~/components/Dot";
import { getLogoForProvider, parseRepo } from "~/utils/helpers/providers";

interface Props {
  repo: string;
  sx?: SxProps;
  children?: React.ReactNode;
  envName?: string;
  color?: "secondary" | "info";
}

export default function AppChip({
  children,
  repo: repoNameWithProvider,
  envName,
  color,
  sx,
}: Props) {
  let repo: string = "";
  let provider: Provider | undefined;

  if (repoNameWithProvider) {
    const data = parseRepo(repoNameWithProvider);
    repo = data.repo;
    provider = data.provider;
  }

  return (
    <Chip
      color={color}
      label={
        <>
          {children && (
            <>
              {children}
              <Dot />
            </>
          )}
          {repo ? (
            <>
              {repo} <Dot />
            </>
          ) : (
            ""
          )}{" "}
          {envName}
        </>
      }
      sx={color ? sx : { bgcolor: "#1F1C3B", ...sx }}
      avatar={provider ? <Avatar src={getLogoForProvider(provider)} /> : <BoltIcon />}
    />
  );
}
