import { useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import ContentCopy from "@mui/icons-material/ContentCopy";
import EmptyPage from "~/components/EmptyPage";

interface Props {
  manifest: Manifest;
  previewEndpoint: string;
  apiPathPrefix: string;
}

interface Endpoint {
  method: string;
  endpoint: string;
  fileName: string;
  curl: string;
}

const methods = ["get", "post", "head", "put", "delete", "patch", "put"];

export default function TabAPI({
  manifest,
  previewEndpoint,
  apiPathPrefix,
}: Props) {
  const [curlCopied, setCurlCopied] = useState<string>();
  const apiEnabled = manifest.apiFiles && manifest.apiFiles?.length > 0;
  const apiFiles: Endpoint[] = useMemo(() => {
    return (
      manifest?.apiFiles?.map(file => {
        const pieces = file.fileName.split(".");
        const methodOrExt = pieces[1]?.toLowerCase();
        const method = methods.find(v => methodOrExt === v) || "ALL";
        const name = pieces[0].replace(/^\//, "").replace(/\[(.*)\]/, ":$1");
        const endpoint = `${apiPathPrefix}/${name}`.replace("/index", "");

        return {
          method,
          endpoint,
          fileName: file.fileName,
          curl: `curl -X ${
            method === "ALL" ? "GET" : method.toUpperCase()
          } "${previewEndpoint.replace(/\/$/, "")}${endpoint}"`,
        };
      }) || []
    );
  }, [manifest?.apiFiles, previewEndpoint]);

  return (
    <Box>
      {apiEnabled ? (
        apiFiles.map(entry => (
          <div
            key={entry.fileName}
            className="bg-blue-10 mb-4 flex w-full text-gray-80"
          >
            <div className="text-white font-bold p-4 bg-black w-20 flex items-center">
              {entry.method.toUpperCase()}
            </div>
            <div className="p-4 flex-1">
              {entry.endpoint}
              <br />
              <span className="text-xs text-gray-50">{entry.fileName}</span>
            </div>
            <div className="p-4 flex items-start">
              <Tooltip
                title={curlCopied === entry.fileName ? "Copied!" : "Copy CURL"}
              >
                <IconButton
                  aria-label={
                    curlCopied === entry.fileName ? "Copied!" : "Copy CURL"
                  }
                  onMouseEnter={() => {
                    setCurlCopied(undefined);
                  }}
                  onClick={() => {
                    navigator.clipboard.writeText(entry.curl);
                    setCurlCopied(entry.fileName);
                  }}
                >
                  <ContentCopy sx={{ fontSize: 16 }} />
                </IconButton>
              </Tooltip>
            </div>
          </div>
        ))
      ) : (
        <EmptyPage sx={{ my: 6 }}>
          <Typography component="span" sx={{ mb: 4, display: "block" }}>
            REST API not detected.
            <br />
            Create a top-level{" "}
            <span className="text-white font-bold">/api</span> folder to get
            started.
          </Typography>
          <Button
            color="secondary"
            variant="contained"
            href="https://www.stormkit.io/docs/features/writing-api"
            rel="noopener noreferrer"
            target="_blank"
          >
            Learn more
          </Button>
        </EmptyPage>
      )}
    </Box>
  );
}
