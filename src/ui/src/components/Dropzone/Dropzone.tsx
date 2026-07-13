import { useEffect, useState } from "react";
import { useDropzone } from "react-dropzone";
import Box from "@mui/material/Box";
import { Typography } from "@mui/material";

interface Props {
  props?: { accept?: string };
  files?: File[];
  onDrop: (acceptedFiles: File[]) => void;
  showDropZone?: boolean; // Whether to show the dropzone area or not
  clickToOpen?: boolean;
  openDialog?: { mode: string };
}

export default function MyDropzone({
  openDialog,
  showDropZone,
  clickToOpen,
  files,
  onDrop,
  props = {},
}: Props) {
  const [otherProps, setOtherProps] = useState<Record<string, any>>(props);
  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    onDrop,
    noClick: true,
  });

  useEffect(() => {
    setOtherProps(
      openDialog?.mode === "folder"
        ? {
            ...otherProps,
            webkitdirectory: "",
            directory: "",
            mozdirectory: "",
          }
        : { ...otherProps }
    );
  }, [openDialog]);

  useEffect(() => {
    openDialog && open();
  }, [otherProps]);

  return (
    <Box {...getRootProps()} sx={{ position: "relative", py: 12 }}>
      {(isDragActive || showDropZone) && (
        <Box
          onClick={clickToOpen ? open : undefined}
          sx={{
            cursor: clickToOpen ? "pointer" : "default",
            position: "absolute",
            left: 0,
            top: 0,
            right: 0,
            bottom: 0,
            border: "3px dashed",
            borderColor: isDragActive
              ? "container.borderContrast"
              : "container.border",
            bgcolor: "container.transparent",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: clickToOpen ? "text.secondary" : "text.primary",
            ":hover": {
              color: clickToOpen ? "text.primary" : undefined,
            },
          }}
        >
          <Typography
            sx={{
              fontSize: 32,
            }}
          >
            {clickToOpen ? "Click or drop here" : "Drop here"}
          </Typography>
          <Typography
            sx={{
              color: "text.secondary",
              position: "absolute",
              bottom: 0,
              mb: 2,
            }}
          >
            {files?.length
              ? `Files: ${files.length}/${files.length}`
              : "No files uploaded yet"}
          </Typography>
        </Box>
      )}
      <input
        {...getInputProps()}
        {...otherProps}
        type="file"
        data-testid="my-dropzone"
      />
    </Box>
  );
}
