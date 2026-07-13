import { useMemo } from "react";
import TextField from "@mui/material/TextField";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Modal from "~/components/Modal";

type TransformerFn = (kv: string[][], separator: string) => string;

interface TextFieldModalProps {
  rows: string[][];
  placeholder?: string;
  isSensitive?: boolean;
  separator: string;
  transformer?: TransformerFn;
  onSave: (value: string) => void;
  onClose: () => void;
}

const transformerFn: TransformerFn = (kv, separator): string => {
  return kv
    .map(k => (k[0] && k[1] ? `${k[0]}${separator}${k[1]}` : ""))
    .join("\n");
};

export default function TextFieldModal({
  rows,
  onClose,
  onSave,
  placeholder,
  separator,
  isSensitive,
  transformer = transformerFn,
}: TextFieldModalProps) {
  const value = useMemo(() => {
    return transformer(rows, separator);
  }, [rows, separator]);

  return (
    <Modal open onClose={onClose}>
      <Box>
        <Box sx={{ p: 2 }}>
          <TextField
            id="key-value-text-area"
            variant="filled"
            multiline
            maxRows={20}
            defaultValue={value}
            placeholder={placeholder}
            fullWidth
            minRows={20}
          />
        </Box>
        <Box sx={{ textAlign: "center", mb: 2 }}>
          <Button
            type="button"
            color="secondary"
            variant="contained"
            onClick={() => {
              const input = document.querySelector(
                "#key-value-text-area"
              ) as HTMLTextAreaElement;

              onSave(input?.value);
              onClose();
            }}
          >
            Set rows
          </Button>
        </Box>
      </Box>
    </Modal>
  );
}
