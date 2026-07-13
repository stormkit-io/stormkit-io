import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import Async from "~/components/Async";

function RedirectToEnvPage() {
  const navigate = useNavigate();
  const { pathname } = useLocation();

  useEffect(() => {
    navigate(`${pathname}/environments/default`, { replace: true });
  }, [navigate]);

  return <></>;
}

export const routes = [
  {
    path: "/environments",
    element: <RedirectToEnvPage />,
  },
  {
    path: "/environments/default",
    element: Async(() => import("~/pages/apps/[id]/environments/Default")),
  },
  {
    path: "/environments/:envId/*",
    element: Async(() => import("~/pages/apps/[id]/environments/[env-id]")),
  },
  {
    path: `/settings`,
    element: Async(() => import("~/pages/apps/[id]/settings")),
  },
  {
    path: `/deployments`,
    element: Async(() => import("~/pages/apps/[id]/deployments")),
  },
  {
    path: "/deployments/:deploymentId",
    element: Async(
      () => import("~/pages/apps/[id]/deployments/[deployment-id]")
    ),
  },
  {
    path: "/feed",
    element: Async(() => import("~/pages/apps/[id]/feed")),
  },
];
