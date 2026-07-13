import { useState } from "react";
import { useNavigate } from "react-router";
import Link from "@mui/material/Link";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import Switch from "@mui/material/Switch";
import FormControlLabel from "@mui/material/FormControlLabel";
import TextField from "@mui/material/TextField";
import ArrowBack from "@mui/icons-material/ArrowBack";
import { deploy } from "~/pages/apps/actions";
import MyDropzone from "~/components/Dropzone";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";

interface Props {
  app: App;
  selected?: Environment;
  toggleModal: (val: boolean) => void;
}

export default function DeployModal({
  toggleModal,
  selected: environment,
  app,
}: Props) {
  const navigate = useNavigate();
  const [files, setFiles] = useState<File[]>([]);
  const [branch, setBranch] = useState<string>(environment?.branch || "");
  const [error, setError] = useState<null | string>(null);
  const [isAutoPublish, setIsAutoPublish] = useState<boolean>(
    environment?.autoPublish || false
  );
  const [loading, setLoading] = useState<boolean>(false);

  const clearForm = () => {
    setBranch("");
    setIsAutoPublish(false);
  };

  return (
    <Modal
      open
      onClose={() => {
        clearForm();
        toggleModal(false);
      }}
    >
      <Card
        component="form"
        error={
          error === "repo-not-found" ? (
            <Typography component="div">
              Repository is inaccessible. See the{" "}
              <Link
                className="font-bold hover:text-white hover:underline"
                href="https://www.stormkit.io/docs/other/troubleshooting#repository-is-innaccessible"
              >
                troubleshooting
              </Link>{" "}
              for help.
            </Typography>
          ) : (
            error && <Typography component="div">{error}</Typography>
          )
        }
        onSubmit={e => {
          e.preventDefault();

          deploy({
            app,
            environment,
            setError,
            setLoading,
            files,
            config: {
              branch,
              publish: isAutoPublish || false,
            },
          }).then(deploy => {
            if (deploy) {
              toggleModal(false);
              navigate(
                `/apps/${app.id}/environments/${environment?.id}/deployments/${deploy.id}`
              );
            }
          });
        }}
      >
        <CardHeader title="Start a deployment" />
        {app.isBare ? (
          <Box sx={{ mb: 4 }}>
            <MyDropzone
              files={files}
              showDropZone
              clickToOpen
              props={{ accept: "application/zip" }}
              onDrop={(acceptedFiles: File[]) => {
                setFiles(acceptedFiles);
              }}
            />
          </Box>
        ) : (
          <Box sx={{ mb: 4 }}>
            <TextField
              name="branch"
              variant="filled"
              label="Checkout branch"
              value={branch}
              onChange={e => {
                setBranch(e.target.value);
              }}
              slotProps={{
                input: {
                  "aria-label": "Branch to deploy",
                },
              }}
              fullWidth
            />
          </Box>
        )}
        <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
          <FormControlLabel
            sx={{ pl: 0, ml: 0 }}
            label="Auto publish"
            control={
              <Switch
                name="autoPublish"
                color="secondary"
                checked={isAutoPublish}
                onChange={e => {
                  setIsAutoPublish(e.target.checked);
                }}
              />
            }
            labelPlacement="start"
          />
          <Typography sx={{ color: "text.secondary" }}>
            When enabled successful deployments will be published automatically
          </Typography>
        </Box>
        <CardFooter sx={{ display: "flex", justifyContent: "space-between" }}>
          <Link
            color="text.secondary"
            href={`/apps/${app.id}/environments/${environment?.id}`}
            sx={{ display: "inline-flex", alignItems: "center" }}
            onClick={() => {
              toggleModal(false);
            }}
          >
            <ArrowBack sx={{ fontSize: 14, mr: 1 }} /> Update build
            configuration
          </Link>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={loading}
          >
            Deploy now
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
