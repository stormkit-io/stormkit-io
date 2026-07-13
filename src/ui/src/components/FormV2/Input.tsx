import React from "react";
import TextField, { TextFieldProps } from "@mui/material/TextField";
import { Tooltip } from "@mui/material";

export type Props = TextFieldProps & {
  tooltip?: React.ReactNode;
};

const Input: React.FC<Props> = ({
  tooltip,
  type = "text",
  variant = "filled",
  autoComplete = "off",
  inputProps,
  InputProps,
  ...props
}) => {
  return (
    <TextField
      type={type}
      color="info"
      variant={variant}
      autoComplete={autoComplete}
      inputProps={{
        sx: { p: 2 },
        ...inputProps,
      }}
      InputProps={{
        ...InputProps,
        endAdornment: (InputProps?.endAdornment || tooltip) && (
          <>
            {InputProps?.endAdornment}
            {tooltip && (
              <Tooltip title={tooltip} arrow className="text-black">
                <span className="fas fa-question-circle" />
              </Tooltip>
            )}
          </>
        ),
      }}
      {...props}
    />
  );
};

export default Input;
