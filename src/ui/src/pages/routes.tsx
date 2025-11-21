import { useEffect } from "react";
import { RouteProps } from "react-router";
import { useLocation, useNavigate } from "react-router-dom";
import Async from "~/components/Async";
import { LocalStorage } from "~/utils/storage";
import { LS_PROVIDER } from "~/utils/api/Api";

type ExtendedRouterProps = RouteProps & {
  cloudOnly?: boolean;
};

function RedirectApps() {
  const navigate = useNavigate();
  const { pathname } = useLocation();

  useEffect(() => {
    navigate(pathname.replace("/app", "/apps"), { replace: true });
  }, [navigate]);

  return <></>;
}

function RedirectNewApp() {
  const navigate = useNavigate();

  useEffect(() => {
    navigate(`/apps/new/${LocalStorage.get(LS_PROVIDER)}`, { replace: true });
  }, [navigate]);

  return <></>;
}

function RedirectToEnvPage() {
  const navigate = useNavigate();
  const { pathname } = useLocation();

  useEffect(() => {
    navigate(`${pathname}/environments/default`, { replace: true });
  }, [navigate]);

  return <></>;
}

function RedirectPaymentSuccess() {
  const navigate = useNavigate();

  useEffect(() => {
    navigate(`/user/account?payment=success`, { replace: true });
  }, [navigate]);

  return <></>;
}

const centerLayout = () => import("~/layouts/CenterLayout");
const teamLayout = () => import("~/layouts/TeamLayout");
const adminLayout = () => import("~/layouts/AdminLayout");

const routes: Array<ExtendedRouterProps> = [
  {
    path: "/",
    element: Async(() => import("~/pages/apps"), teamLayout),
  },
  {
    path: "/clone",
    element: Async(() => import("~/pages/clone"), centerLayout),
  },
  {
    path: "/auth",
    element: Async(() => import("~/pages/auth"), centerLayout),
  },
  {
    path: "/logout",
    element: Async(() => import("~/pages/logout")),
  },
  {
    path: "/apps/new",
    element: <RedirectNewApp />,
  },
  {
    path: "/apps/new/github",
    element: Async(() => import("~/pages/apps/new/github"), centerLayout),
  },
  {
    path: "/apps/new/gitlab",
    element: Async(() => import("~/pages/apps/new/gitlab"), centerLayout),
  },
  {
    path: "/apps/new/bitbucket",
    element: Async(() => import("~/pages/apps/new/bitbucket"), centerLayout),
  },
  {
    path: "/apps/new/url",
    element: Async(() => import("~/pages/apps/new/url"), centerLayout),
  },
  {
    path: "/app/invitation/accept",
    element: Async(() => import("~/pages/app/invitation/Accept")),
  },
  {
    path: "/app/:id/*",
    element: <RedirectApps />,
  },
  {
    path: "/apps/:id",
    element: <RedirectToEnvPage />,
  },
  {
    path: "/apps/:id/*",
    element: Async(() => import("~/layouts/AppLayout")),
  },
  {
    path: "/user/account",
    element: Async(() => import("~/pages/user/account"), centerLayout),
  },
  {
    path: "/user/payment/success",
    element: <RedirectPaymentSuccess />,
  },
  {
    path: "/invitation/accept",
    element: Async(() => import("~/pages/team/InvitationAccept"), centerLayout),
  },
  {
    path: "/admin",
    element: Async(() => import("~/pages/admin"), adminLayout),
  },
  {
    path: "/admin/system",
    element: Async(() => import("~/pages/admin/System"), adminLayout),
  },
  {
    path: "/admin/jobs",
    element: Async(() => import("~/pages/admin/Jobs"), adminLayout),
  },
  {
    path: "/admin/subscription",
    element: Async(() => import("~/pages/admin/Subscription"), adminLayout),
  },
  {
    path: "/admin/proxies",
    element: Async(() => import("~/pages/admin/Proxies"), adminLayout),
  },
  {
    path: "/admin/cloud/apps",
    element: Async(() => import("~/pages/admin/CloudApps"), adminLayout),
    cloudOnly: true,
  },
  {
    path: "/admin/git",
    element: Async(() => import("~/pages/admin/Git"), adminLayout),
  },
  {
    path: "/admin/auth",
    element: Async(() => import("~/pages/admin/AuthConfig"), adminLayout),
  },
  {
    path: "/:team",
    element: Async(() => import("~/pages/apps"), teamLayout),
  },
  {
    path: "/:team/settings",
    element: Async(() => import("~/pages/team/Settings"), teamLayout),
  },
  {
    path: "/:team/feed",
    element: Async(() => import("~/pages/team/feed"), teamLayout),
  },
  {
    path: "/:team/insights",
    element: Async(() => import("~/pages/team/insights"), teamLayout),
  },
  {
    path: "/:team/deployments",
    element: Async(() => import("~/pages/team/deployments"), teamLayout),
  },
  {
    path: "/*",
    element: Async(() => import("~/components/Errors/Error404"), centerLayout),
  },
];

export default routes;
