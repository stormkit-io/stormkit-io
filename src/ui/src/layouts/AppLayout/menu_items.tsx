import SettingsIcon from "@mui/icons-material/Settings";
import RocketLaunchIcon from "@mui/icons-material/RocketLaunch";
import CodeIcon from "@mui/icons-material/Code";
import ScheduleIcon from "@mui/icons-material/Schedule";
import StorageIcon from "@mui/icons-material/Storage";
import TextSnippetIcon from "@mui/icons-material/TextSnippet";
import InsertChartIcon from "@mui/icons-material/InsertChart";
import MailIcon from "@mui/icons-material/Mail";
import GroupIcon from "@mui/icons-material/Group";
import DatabaseIcon from "@mui/icons-material/Inventory";
import { SxProps } from "@mui/material/styles";

export const appMenuItems = ({
  app,
  pathname,
}: {
  app: App;
  pathname: string;
}): Path[] => [
  {
    // List settings
    path: `/apps/${app.id}/feed`,
    text: "Activity Feed",
    isActive: pathname.endsWith("/feed"),
  },
  {
    // List settings
    path: `/apps/${app.id}/settings`,
    text: "Settings",
    isActive: pathname.endsWith("/settings"),
  },
];

interface Path {
  path: string;
  icon?: React.ReactNode;
  text: React.ReactNode;
  isActive?: boolean;
  children?: Omit<Path, "children">[];
}

const Icon = (Icon: any, sx: SxProps = {}) => {
  return <Icon sx={{ fontSize: 15, mr: 2, color: "text.secondary", ...sx }} />;
};

export const envMenuItems = ({
  app,
  env,
  pathname,
}: {
  app: App;
  env: Environment;
  pathname: string;
}): Path[] => {
  if (!env) {
    return [];
  }

  const envPath = `/apps/${app.id}/environments/${env.id}`;

  const items: Path[] = [
    {
      text: "Config",
      path: envPath,
      isActive: pathname === envPath,
      icon: Icon(SettingsIcon),
    },
    {
      path: `${envPath}/deployments`,
      text: "Deployments",
      icon: Icon(RocketLaunchIcon),
      isActive:
        pathname.includes("/deployments") && !pathname.includes("runtime-logs"),
    },
    {
      text: "Snippets",
      path: `${envPath}/snippets`,
      icon: Icon(CodeIcon),
      isActive: pathname.includes("/snippets"),
    },
    {
      text: "Triggers",
      path: `${envPath}/function-triggers`,
      icon: Icon(ScheduleIcon),
      isActive: pathname.includes("/function-triggers"),
    },
    {
      text: "Volumes",
      path: `${envPath}/volumes`,
      icon: Icon(StorageIcon),
      isActive: pathname.includes("/volumes"),
    },
    {
      text: "Database",
      path: `${envPath}/database`,
      icon: Icon(DatabaseIcon),
      isActive: pathname.includes("/database"),
    },
  ];

  if (env.build?.vars?.["SK_AUTH_ENABLED"] === "true") {
    items.push({
      text: "Authentication",
      path: `${envPath}/auth`,
      icon: Icon(GroupIcon),
      isActive: pathname.includes("/auth"),
      children: [
        {
          text: "Providers",
          path: `${envPath}/auth`,
          isActive: pathname.endsWith("/auth"),
        },
        {
          text: "Users",
          path: `${envPath}/auth/users`,
          isActive: pathname.endsWith("/auth/users"),
        },
      ],
    });
  }

  items.push({
    text: "Mailer",
    path: `${envPath}/mailer`,
    icon: Icon(MailIcon),
    isActive: pathname.includes("/mailer"),
  });

  if (env.published?.length) {
    items.push({
      text: "Runtime logs",
      path: `${envPath}/deployments/${env.published[0].deploymentId}/runtime-logs`,
      icon: Icon(TextSnippetIcon),
      isActive: pathname.includes("/runtime-logs"),
    });
  }

  items.push({
    text: "Analytics",
    path: `${envPath}/analytics`,
    icon: Icon(InsertChartIcon),
    isActive: pathname.includes("/analytics"),
  });

  return items;
};
