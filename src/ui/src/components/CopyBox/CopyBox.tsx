import type { TextFieldProps } from "@mui/material/TextField";
import { useState } from "react";
import Tooltip from "@mui/material/Tooltip";
import TextField from "@mui/material/TextField";
import IconButton from "@mui/material/IconButton";
import CheckIcon from "@mui/icons-material/Check";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";

let id = 0;

// copyToClipboard prefers the async Clipboard API, which is the reliable path on
// modern browsers and secure contexts. It falls back to selecting the input and
// the legacy execCommand for older browsers or non-secure (http) contexts where
// navigator.clipboard is unavailable.
function copyToClipboard(value: string, inputId: string) {
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(value).catch(() => selectAndExec(inputId));
    return;
  }

  selectAndExec(inputId);
}

function selectAndExec(inputId: string) {
  const input = document.querySelector<HTMLInputElement>(`#${inputId}`);

  if (!input) {
    return;
  }

  input.focus();
  input.select();
  document.execCommand("copy");
}

export default function CopyBox({
  value,
  slotProps,
  sx,
  ...rest
}: TextFieldProps) {
  const [clicked, setClicked] = useState(false);
  const inputId = `copy-token-${id++}`;

  return (
    <TextField
      id={inputId}
      value={value}
      fullWidth
      aria-label="Copy content"
      {...rest}
      sx={{
        bgcolor: "rgba(0,0,0,0.2)",
        borderRadius: 1,
        ...sx,
      }}
      slotProps={{
        ...slotProps,
        htmlInput: {
          ...(slotProps?.htmlInput ?? {}),
          readOnly: true,
        },
        input: {
          ...slotProps?.input,
          endAdornment: (
            <Tooltip open={clicked} title="Copied to clipboard" arrow>
              <IconButton
                type="button"
                aria-label="Copy to clipboard"
                onClick={() => {
                  copyToClipboard(String(value ?? ""), inputId);
                  setClicked(true);
                  setTimeout(() => {
                    setClicked(false);
                  }, 2000);
                }}
              >
                {clicked ? (
                  <CheckIcon sx={{ fontSize: 18 }} />
                ) : (
                  <ContentCopyIcon sx={{ fontSize: 18 }} />
                )}
              </IconButton>
            </Tooltip>
          ),
        },
      }}
    />
  );
}
