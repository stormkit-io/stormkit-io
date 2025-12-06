import type { RouteProps } from "react-router";
import Async from "~/components/Async";

const routes: Array<RouteProps> = [
  {
    path: "/",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/config")
    ),
  },
  {
    path: "/function-triggers",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/function-triggers")
    ),
  },
  {
    path: "/function-triggers/:triggerId/logs",
    element: Async(
      () =>
        import(
          "~/pages/apps/[id]/environments/[env-id]/function-triggers/[trigger-id]"
        )
    ),
  },
  {
    path: "/snippets",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/snippets")
    ),
  },
  {
    path: "/volumes",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/volumes")
    ),
  },
  {
    path: "/database",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/database")
    ),
  },
  {
    path: "/deployments",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/deployments")
    ),
  },
  {
    path: "/deployments/:deploymentId",
    element: Async(
      () =>
        import(
          "~/pages/apps/[id]/environments/[env-id]/deployments/[deployment-id]"
        )
    ),
  },
  {
    path: "/deployments/:deploymentId/runtime-logs",
    element: Async(
      () =>
        import(
          "~/pages/apps/[id]/environments/[env-id]/deployments/runtime-logs"
        )
    ),
  },
  {
    path: "/analytics",
    element: Async(
      () => import("~/pages/apps/[id]/environments/[env-id]/analytics")
    ),
  },
];

export default routes;
