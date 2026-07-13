import { useEffect, useState } from "react";
import { useDropzone } from "react-dropzone";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import DeleteIcon from "@mui/icons-material/Delete";
import VisibilityIcon from "@mui/icons-material/Visibility";
import CloudDownloadIcon from "@mui/icons-material/CloudDownload";
import LockOpenIcon from "@mui/icons-material/LockOpen";
import LockIcon from "@mui/icons-material/Lock";
import ConfirmModal from "~/components/ConfirmModal";
import CardRow from "~/components/CardRow";
import { formattedDate } from "~/utils/helpers/deployments";
import {
  uploadFiles,
  removeFiles,
  useFetchFiles,
  fetchDownloadUrl,
  toggleVisibility,
} from "./actions";
import EmptyList from "~/components/EmptyPage";

interface Props {
  envId: string;
  appId: string;
  loading?: boolean;
  setError: (v?: React.ReactNode) => void;
  setLoading: (v: boolean) => void;
  openDialog?: { mode: string };
}

export default function VolumesDropzone({
  envId,
  appId,
  openDialog,
  loading,
  setLoading,
  setError,
}: Props) {
  const props = { appId, envId, setError, setLoading };
  const [refreshToken, setRefreshToken] = useState(0);
  const [fileToDelete, setFileToDelete] = useState<VolumeFile>();
  const [fileToToggle, setFileToToggle] = useState<VolumeFile>();
  const [otherProps, setOtherProps] = useState({});
  const { files } = useFetchFiles({ refreshToken, ...props });
  const { onDrop } = uploadFiles({ setRefreshToken, ...props });
  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    onDrop,
    noClick: true,
  });

  useEffect(() => {
    setOtherProps(
      openDialog?.mode === "folder"
        ? { webkitdirectory: "", directory: "", mozdirectory: "" }
        : {}
    );
  }, [openDialog]);

  useEffect(() => {
    openDialog && open();
  }, [otherProps]);

  const isEmpty = !loading && files.length === 0;

  return (
    <Box
      {...getRootProps()}
      sx={{ position: "relative", py: isEmpty ? 12 : 0 }}
    >
      {isDragActive && (
        <Box
          sx={{
            position: "absolute",
            left: 0,
            top: 0,
            right: 0,
            bottom: 0,
            border: `5px dashed`,
            borderColor: "container.border",
            bgcolor: "container.transparent",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: 32,
          }}
        >
          Drop here...
        </Box>
      )}
      <input
        {...getInputProps()}
        {...otherProps}
        type="file"
        data-testid="volumes-dropzone"
      />
      {isEmpty && <EmptyList sx={{ my: 0 }}>No files uploaded yet</EmptyList>}
      {files.map(file => (
        <CardRow
          key={file.id}
          menuItems={[
            {
              text: `Make ${file.isPublic ? " private" : " public"}`,
              icon: <VisibilityIcon fontSize="small" />,
              onClick() {
                setFileToToggle(file);
              },
            },
            {
              text: "Download",
              icon: <CloudDownloadIcon fontSize="small" />,
              onClick() {
                fetchDownloadUrl({ appId, envId, fileId: file.id })
                  .then(url => {
                    const link = document.createElement("a");
                    link.href = url;
                    link.setAttribute("download", file.name);

                    document.body.appendChild(link);
                    link.click();
                    document.parentNode?.removeChild(link);
                  })
                  .catch(() => {
                    setError(
                      `Error while generating download URL for file: ${file.name}`
                    );
                  });
              },
            },
            {
              text: "Delete",
              icon: <DeleteIcon fontSize="small" />,
              onClick() {
                setFileToDelete(file);
              },
            },
          ]}
        >
          <Typography sx={{ display: "flex", alignItems: "center" }}>
            {file.isPublic ? (
              <LockOpenIcon color="warning" fontSize="small" />
            ) : (
              <LockIcon color="success" fontSize="small" />
            )}
            <Typography component="span" sx={{ flex: 1, ml: 1 }}>
              {file.name}
              {file.publicLink && (
                <>
                  {" "}
                  <br />
                  <Link
                    target="_blank"
                    rel="noreferrer noopener"
                    color="text.secondary"
                    href={file.publicLink}
                    sx={{
                      display: "inline-block",
                      textOverflow: "ellipsis",
                      overflow: "hidden",
                      whiteSpace: "pre",
                      maxWidth: { xs: "200px", md: "400px", lg: "600px" },
                    }}
                  >
                    {file.publicLink}
                  </Link>
                </>
              )}
            </Typography>
            <Typography component="span" sx={{ fontSize: 11 }}>
              {formattedDate(file.updatedAt || file.createdAt)}
            </Typography>
          </Typography>
        </CardRow>
      ))}
      {fileToDelete && (
        <ConfirmModal
          onCancel={() => {
            setFileToDelete(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            removeFiles({ appId, envId, fileId: fileToDelete.id })
              .then(() => {
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setError("Something went wrong while deleting the file.");
              })
              .finally(() => {
                setLoading(false);
                setFileToDelete(undefined);
              });

            return "";
          }}
        >
          <>
            This will remove{" "}
            <Typography
              component="span"
              color="text.secondary"
              fontSize="medium"
            >
              {fileToDelete.name}
            </Typography>{" "}
            completely and it will no longer be accessible.
          </>
        </ConfirmModal>
      )}
      {fileToToggle && (
        <ConfirmModal
          onCancel={() => {
            setFileToToggle(undefined);
          }}
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            setError(null);

            toggleVisibility({
              appId,
              envId,
              fileId: fileToToggle.id,
              visibility: fileToToggle.isPublic ? "private" : "public",
            })
              .then(() => {
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setError(
                  "Something went wrong while toggling visibility of the file."
                );
              })
              .finally(() => {
                setLoading(false);
                setFileToToggle(undefined);
              });

            return "";
          }}
        >
          <>
            This will make{" "}
            <Typography
              component="span"
              color="text.secondary"
              fontSize="medium"
            >
              {fileToToggle.name}
            </Typography>{" "}
            {fileToToggle.isPublic
              ? "a private file. Only authenticated users will be able to access the file."
              : "a public file. Anyone with the link will be able to see the file."}{" "}
          </>
        </ConfirmModal>
      )}
    </Box>
  );
}
