import SettingsIcon from "@mui/icons-material/Settings";
import RocketLaunchIcon from "@mui/icons-material/RocketLaunch";
import CodeIcon from "@mui/icons-material/Code";
import ScheduleIcon from "@mui/icons-material/Schedule";
import StorageIcon from "@mui/icons-material/Storage";
import TextSnippetIcon from "@mui/icons-material/TextSnippet";
import InsertChartIcon from "@mui/icons-material/InsertChart";
import DatabaseIcon from "@mui/icons-material/Inventory";

export const appMenuItems = ({
  app,
  pathname,
}: {
  app: App;
  pathname: string;
}): Path[] => [
  {
    // List environments
    path: `/apps/${app.id}/environments`,
    text: "Environments",
    isActive: pathname.includes("/environments"),
  },
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
}

const Icon = (Icon: any) => {
  return <Icon sx={{ fontSize: 15, mr: 2, color: "text.secondary" }} />;
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

  const items = [
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
