import { useContext, useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import SettingsIcon from "@mui/icons-material/Settings";
import UploadFileIcon from "@mui/icons-material/UploadFile";
import UploadFolderIcon from "@mui/icons-material/DriveFolderUpload";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { AuthContext } from "~/pages/auth/Auth.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import EmptyPage from "~/components/EmptyPage";
import CardFooter from "~/components/CardFooter";
import VolumesConfigModal from "./VolumesConfigModal";
import VolumesDropzone from "./VolumesDropzone";
import { useFetchConfig } from "./actions";

function LearnMoreButton() {
  return (
    <Button
      href="https://www.stormkit.io/docs/features/volumes"
      variant="outlined"
      color="primary"
      target="_blank"
      rel="noreferrer noopener"
      endIcon={<OpenInNewIcon />}
    >
      Learn more
    </Button>
  );
}

interface AdminEmptyViewProps {
  onConfigureClick: () => void;
}

function AdminEmptyView({ onConfigureClick }: AdminEmptyViewProps) {
  return (
    <EmptyPage>
      <Typography
        component="span"
        variant="h6"
        sx={{ mb: 4, display: "block" }}
      >
        Persist your files seamlessly using Stormkit Volumes
      </Typography>
      <Box component="span" sx={{ display: "block" }}>
        <LearnMoreButton />
        <Button
          variant="contained"
          color="secondary"
          sx={{ ml: 2 }}
          onClick={onConfigureClick}
        >
          Configure
        </Button>
      </Box>
    </EmptyPage>
  );
}

function UserEmptyView() {
  return (
    <EmptyPage>
      <>
        <Typography
          component="span"
          variant="h6"
          sx={{ mb: 4, display: "block" }}
        >
          Volumes is not configured for this Stormkit instance.
          <br />
          Contact your administrator for more information.
        </Typography>
        <LearnMoreButton />
      </>
    </EmptyPage>
  );
}

interface ActionButtonProps {
  mr?: number;
  icon: React.ReactNode;
  onClick: () => void;
  text: string;
}

function ActionButton({ mr, icon, onClick, text }: ActionButtonProps) {
  return (
    <Button
      variant="outlined"
      size="small"
      sx={{
        mr,
        "> .MuiButton-icon": { mr: { xs: 0, md: 0.75 } },
      }}
      startIcon={icon}
      onClick={onClick}
    >
      <Box component="span" sx={{ display: { xs: "none", md: "inline" } }}>
        {text}
      </Box>
    </Button>
  );
}

export default function Volumes() {
  const [refreshToken, setRefreshToken] = useState<number>();
  const { user } = useContext(AuthContext);
  const { app } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const { config, loading, error, paymentRequired } = useFetchConfig({
    refreshToken,
  });
  const [uploadErr, setUploadErr] = useState<React.ReactNode>();
  const [uploading, setUploading] = useState(false);
  const [success, setSuccess] = useState<string>();
  const [isConfigModalOpen, setIsConfigModalOpen] = useState(false);
  const [upload, setUpload] = useState<{ mode: string }>();

  // If the request is not loading and there is no error,
  // and the configuration object is not null, then it's configured.
  const isConfigured = useMemo(
    () => !loading && !error && Boolean(config),
    [config, loading, error]
  );

  if (paymentRequired) {
    return (
      <Card sx={{ width: "100%" }}>
        <CardHeader
          title="Volumes"
          subtitle="Manage persisted files across your application"
        />
        <EmptyPage paymentRequired />
      </Card>
    );
  }

  return (
    <Card
      success={success}
      successTitle={false}
      onSuccessClose={() => setSuccess(undefined)}
      error={error || uploadErr}
      errorTitle={uploadErr ? "Upload failed:" : undefined}
      loading={loading || uploading}
      contentPadding={false}
      sx={{ width: "100%" }}
    >
      <CardHeader
        title="Volumes"
        subtitle="Manage persisted files across your application"
        actions={
          isConfigured && (
            <>
              <ActionButton
                mr={user?.isAdmin ? 2 : 0}
                onClick={() => setUpload({ mode: "file" })}
                icon={<UploadFileIcon />}
                text="Upload file"
              />

              <ActionButton
                mr={user?.isAdmin ? 2 : 0}
                onClick={() => setUpload({ mode: "folder" })}
                icon={<UploadFolderIcon />}
                text="Upload folder"
              />
              {user?.isAdmin && (
                <ActionButton
                  icon={<SettingsIcon />}
                  onClick={() => setIsConfigModalOpen(true)}
                  text="Configure"
                />
              )}
            </>
          )
        }
      />
      {isConfigured ? (
        <VolumesDropzone
          appId={app.id}
          envId={environment.id!}
          loading={loading}
          openDialog={upload}
          setLoading={setUploading}
          setError={setUploadErr}
        />
      ) : user?.isAdmin ? (
        <>
          <AdminEmptyView onConfigureClick={() => setIsConfigModalOpen(true)} />
        </>
      ) : (
        <UserEmptyView />
      )}
      {isConfigured && <CardFooter>&nbsp;</CardFooter>}
      {isConfigModalOpen && (
        <VolumesConfigModal
          config={config!}
          onSuccess={() => {
            setRefreshToken(Date.now());
            setIsConfigModalOpen(false);
            setSuccess("Volumes configuration was saved successfully");
          }}
          onClose={() => setIsConfigModalOpen(false)}
        />
      )}
    </Card>
  );
}
