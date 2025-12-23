import React, { useState } from "react";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Button from "@mui/material/Button";
import ArrowForward from "@mui/icons-material/ArrowForward";
import AssignmentTurnedIn from "@mui/icons-material/AssignmentTurnedIn";
import { formattedDate } from "~/utils/helpers/deployments";
import Dot from "~/components/Dot";
import CardRow from "~/components/CardRow";
import DiffModal from "./DiffModal";

interface Props {
  audit: Audit;
  children?: React.ReactNode;
  forceDiff?: boolean;
}

interface LinkProps {
  audit: Audit;
  hash?: string;
  children?: React.ReactNode;
}

const plural = (singular: string, plural: string, count?: number) => {
  return count === 1 ? `1 ${singular}` : `${count || 0} ${plural}`;
};

function EnvLink({ audit, children, hash }: LinkProps) {
  return (
    <Link
      href={`/apps/${audit.appId}/environments/${audit.envId}${hash || ""}`}
      sx={{
        textDecoration: "underline",
        ":hover": { textDecoration: "underline" },
      }}
    >
      {children || audit.envName || "environment"}
    </Link>
  );
}

function AppLink({ audit, children }: LinkProps) {
  return <Link href={`/apps/${audit.appId}`}>{children || "application"}</Link>;
}

function AuditRow({ audit, children, forceDiff }: Props) {
  const [showModal, setShowModal] = useState(false);
  const hasDiff =
    (audit.diff?.old &&
      audit.diff?.new &&
      Object.keys(audit.diff.old).length > 0 &&
      Object.keys(audit.diff.new).length > 0) ||
    forceDiff;

  return (
    <CardRow
      key={audit.id}
      icon={
        <AssignmentTurnedIn sx={{ fontSize: 18, color: "text.secondary" }} />
      }
      actions={
        hasDiff ? (
          <Button
            variant="text"
            endIcon={<ArrowForward sx={{ fontSize: "0.9rem !important" }} />}
            onClick={() => {
              setShowModal(true);
            }}
          >
            Diff
          </Button>
        ) : (
          ""
        )
      }
    >
      <Typography component="div" sx={{ display: "block" }}>
        {children}
        <Box sx={{ fontSize: 11.5, color: "text.secondary", mt: 0.5 }}>
          by {audit.userDisplay || `${audit.tokenName} api key`}
          <Dot />
          {formattedDate(audit.timestamp).toLocaleLowerCase()}
        </Box>
      </Typography>
      {showModal && (
        <DiffModal
          audit={audit}
          onClose={() => {
            setShowModal(false);
          }}
        />
      )}
    </CardRow>
  );
}

export default function AuditMessage({ audit }: Props) {
  switch (audit.action) {
    case "CREATE:DOMAIN":
      return (
        <AuditRow audit={audit}>
          Added {audit.diff.new.domainName} domain to <EnvLink audit={audit} />{" "}
          environment
        </AuditRow>
      );

    case "UPDATE:DOMAIN":
      if (audit.diff.new.domainCertKey) {
        return (
          <AuditRow audit={audit}>
            Added custom certificate to{" "}
            <EnvLink audit={audit} hash="#domains">
              {audit.diff.old.domainName}
            </EnvLink>
          </AuditRow>
        );
      }

      if (audit.diff.old.domainCertKey) {
        return (
          <AuditRow audit={audit}>
            Removed custom certificate from{" "}
            <EnvLink audit={audit} hash="#domains">
              {audit.diff.old.domainName}
            </EnvLink>
          </AuditRow>
        );
      }

    case "DELETE:DOMAIN":
      return (
        <AuditRow audit={audit}>
          Removed {audit.diff.old.domainName} domain from{" "}
          <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "UPDATE:ENV":
      return (
        <AuditRow audit={audit}>
          Updated the <EnvLink audit={audit}>{audit.diff.new.envName}</EnvLink>{" "}
          environment
        </AuditRow>
      );

    case "CREATE:ENV":
      return (
        <AuditRow audit={audit}>
          Created the <EnvLink audit={audit}>{audit.diff.new.envName}</EnvLink>{" "}
          environment
        </AuditRow>
      );

    case "DELETE:ENV":
      return (
        <AuditRow audit={audit}>
          Removed the <EnvLink audit={audit}>{audit.diff.old.envName}</EnvLink>{" "}
          environment
        </AuditRow>
      );

    case "CREATE:APP":
      return (
        <AuditRow audit={audit}>
          Created the <AppLink audit={audit}>{audit.diff.new.appName}</AppLink>{" "}
          application
        </AuditRow>
      );

    case "UPDATE:APP":
      return (
        <AuditRow audit={audit}>
          Updated the <AppLink audit={audit}>{audit.diff.new.appName}</AppLink>{" "}
          application
        </AuditRow>
      );

    case "DELETE:APP":
      return (
        <AuditRow audit={audit}>
          Deleted the {audit.diff.old.appName} application
        </AuditRow>
      );

    case "CREATE:SNIPPET":
      return (
        <AuditRow audit={audit} forceDiff>
          Created{" "}
          {plural(
            "new snippet",
            "new snippets",
            audit.diff.new.snippets?.length
          )}{" "}
          in <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "UPDATE:SNIPPET":
      return (
        <AuditRow audit={audit}>
          Updated 1 snippet in <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "DELETE:SNIPPET":
      return (
        <AuditRow audit={audit}>
          Deleted{" "}
          {plural(
            "snippet",
            "snippets",
            audit.diff.new?.snippets?.length || audit.diff.old?.snippets?.length // Backwards compatibility. It should be old.snippets.
          )}{" "}
          in <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "UPDATE:AUTHWALL":
      return (
        <AuditRow audit={audit} forceDiff>
          {audit.diff.new.authWallStatus === "off" ? "Disabled" : "Enabled"}{" "}
          auth wall in <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "CREATE:AUTHWALL":
      return (
        <AuditRow audit={audit} forceDiff>
          Created new auth login for <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "DELETE:AUTHWALL":
      return (
        <AuditRow audit={audit} forceDiff>
          Deleted auth login from <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "CREATE:SCHEMA":
      return (
        <AuditRow audit={audit} forceDiff>
          Created schema for <EnvLink audit={audit} /> environment
        </AuditRow>
      );

    case "DELETE:SCHEMA":
      return (
        <AuditRow audit={audit} forceDiff>
          Deleted schema from <EnvLink audit={audit} /> environment
        </AuditRow>
      );
  }

  return audit.action;
}
