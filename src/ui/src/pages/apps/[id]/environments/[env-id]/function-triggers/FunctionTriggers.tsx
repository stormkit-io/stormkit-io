import { useContext, useState } from "react";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/ModeEdit";
import PlayIcon from "@mui/icons-material/PlayArrow";
import TimeIcon from "@mui/icons-material/AccessTime";
import CircleIcon from "@mui/icons-material/Circle";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Tooltip from "@mui/material/Tooltip";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import ConfirmModal from "~/components/ConfirmModal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import EmptyPage from "~/components/EmptyPage";
import Span from "~/components/Span";
import FunctionTriggerModal from "./FunctionTriggerModal";
import TriggerLogsDrawer from "./_components/TriggerLogsDrawer";
import * as actions from "./actions";

const {
  useFetchFunctionTriggers,
  deleteFunctionTrigger,
  invokeFunctionTrigger,
} = actions;

const colors: Record<
  FunctionTriggerMethod,
  "warning" | "info" | "error" | "primary" | "success"
> = {
  POST: "warning",
  PUT: "info",
  DELETE: "error",
  GET: "success",
  PATCH: "primary",
};

function nextRun(time: number) {
  return new Date(time).toLocaleDateString("en", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function FunctionTriggers() {
  const { app } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const [toBeModified, setToBeModified] = useState<FunctionTrigger>();
  const [toBeDeleted, setToBeDeleted] = useState<FunctionTrigger>();
  const [isFunctionTriggerModalOpen, setFunctionTriggerModal] = useState(false);
  const [actionError, setActionError] = useState<string>();
  const [invoking, setInvoking] = useState<string>();
  const [drawer, setDrawer] = useState<{
    trigger: FunctionTrigger;
    initialLog?: TriggerLog;
  }>();
  // Kept separate from `drawer` so the panel stays mounted while it slides out.
  const [isDrawerOpen, setDrawerOpen] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  const { error, loading, functionTriggers, paymentRequired } =
    useFetchFunctionTriggers({
      appId: app.id,
      environmentId: environment.id!,
      refreshToken,
    });

  const openDrawer = (trigger: FunctionTrigger, initialLog?: TriggerLog) => {
    setDrawer({ trigger, initialLog });
    setDrawerOpen(true);
  };

  const handleDelete = ({
    setError,
    setLoading,
  }: {
    setError: SetError;
    setLoading: SetLoading;
  }) => {
    setLoading(true);

    deleteFunctionTrigger({
      tfid: toBeDeleted?.id!,
      appId: app.id,
      envId: environment.id!,
    })
      .then(() => {
        setRefreshToken(Date.now());
      })
      .catch(res => {
        setError(
          typeof res === "string"
            ? res
            : "Something went wrong while deleting trigger."
        );
      })
      .finally(() => {
        setToBeDeleted(undefined);
        setLoading(false);
      });
  };

  const handleInvoke = (f: FunctionTrigger) => {
    setActionError(undefined);
    setInvoking(f.id);

    invokeFunctionTrigger({
      tfid: f.id!,
      appId: app.id,
      envId: environment.id!,
    })
      .then(({ log }) => {
        openDrawer(f, log);
      })
      .catch(res => {
        setActionError(
          typeof res === "string"
            ? res
            : "Something went wrong while running the trigger."
        );
      })
      .finally(() => {
        setInvoking(undefined);
      });
  };

  if (paymentRequired) {
    return (
      <Card
        sx={{ width: "100%" }}
        loading={loading}
        error={error}
        contentPadding={false}
      >
        <CardHeader
          title="Periodic Triggers"
          subtitle="Send periodic requests to your endpoints."
        />
        <EmptyPage paymentRequired />
      </Card>
    );
  }

  return (
    <Card
      sx={{ width: "100%" }}
      loading={loading}
      error={error || actionError}
      contentPadding={false}
    >
      <CardHeader
        title="Periodic Triggers"
        subtitle="Send periodic requests to your endpoints."
      />
      {functionTriggers?.map((f, i) => (
        <CardRow
          key={f.id}
          menuItems={[
            {
              icon: <PlayIcon />,
              text: invoking === f.id ? "Running…" : "Trigger now",
              disabled: invoking === f.id,
              onClick: () => handleInvoke(f),
            },
            {
              icon: <TimeIcon />,
              text: "Past triggers",
              onClick: () => openDrawer(f),
            },
            {
              icon: <EditIcon />,
              text: "Modify",
              onClick: () => {
                setToBeModified(f);
                setFunctionTriggerModal(true);
              },
            },
            {
              icon: <DeleteIcon />,
              text: "Delete",
              onClick: () => {
                setToBeModified(undefined);
                setToBeDeleted(f);
              },
            },
          ]}
        >
          <Typography
            sx={{ display: "flex", alignItems: "center", width: "100%" }}
          >
            <Chip
              size="small"
              component="span"
              color={colors[f.options.method] || "primary"}
              label={f.options.method}
              sx={{ fontSize: 10, mr: 2, minWidth: "60px" }}
            />
            <Tooltip title="See past triggers">
              <Typography
                component="span"
                color="text.secondary"
                noWrap
                data-testid="trigger-url"
                onClick={() => openDrawer(f)}
                sx={{
                  mr: 2,
                  flex: 1,
                  cursor: "pointer",
                  "&:hover": { textDecoration: "underline" },
                }}
              >
                {f.options.url}
              </Typography>
            </Tooltip>
            <Span sx={{ display: "inline-flex", alignItems: "center" }}>
              <Tooltip
                title={
                  f.status &&
                  f.nextRunAt && (
                    <Typography component="span" sx={{ fontSize: 12 }}>
                      Next run at {nextRun(f.nextRunAt * 1000)}
                    </Typography>
                  )
                }
              >
                <span style={{ display: "inline-flex", alignItems: "center" }}>
                  <TimeIcon sx={{ mr: 1, fontSize: 16 }} />
                  {f.cron}
                </span>
              </Tooltip>
            </Span>
            <Span sx={{ display: "inline-flex", alignItems: "center" }}>
              <CircleIcon
                color={f.status ? "success" : "error"}
                sx={{ fontSize: 10, mr: 1 }}
              />
              {f.status ? "Enabled" : "Disabled"}
            </Span>
          </Typography>
        </CardRow>
      ))}
      {!loading && !error && !functionTriggers?.length && (
        <EmptyPage>
          <>
            It's quite empty in here.
            <br />
            Create a new trigger to call your functions periodically.
          </>
        </EmptyPage>
      )}
      <CardFooter sx={{ textAlign: "center" }}>
        <Button
          type="button"
          variant="contained"
          color="secondary"
          onClick={() => {
            setFunctionTriggerModal(true);
          }}
        >
          New trigger
        </Button>
      </CardFooter>
      {isFunctionTriggerModalOpen && (
        <FunctionTriggerModal
          app={app}
          environment={environment}
          triggerFunction={toBeModified}
          closeModal={() => {
            setToBeModified(undefined);
            setFunctionTriggerModal(false);
          }}
          onSuccess={() => setRefreshToken(Date.now())}
        />
      )}
      {drawer && (
        <TriggerLogsDrawer
          key={`${drawer.trigger.id}-${drawer.initialLog?.createdAt ?? ""}`}
          open={isDrawerOpen}
          trigger={drawer.trigger}
          appId={app.id}
          envId={environment.id!}
          initialLog={drawer.initialLog}
          onClose={() => setDrawerOpen(false)}
          onExited={() => setDrawer(undefined)}
        />
      )}
      {toBeDeleted && functionTriggers && (
        <ConfirmModal
          onConfirm={handleDelete}
          onCancel={() => setToBeDeleted(undefined)}
        >
          This will delete the trigger immediately.
        </ConfirmModal>
      )}
    </Card>
  );
}
