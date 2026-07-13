import React from "react";
import TextField, { TextFieldProps } from "@mui/material/TextField";
import { Tooltip } from "@mui/material";

type Props = TextFieldProps & {
  tooltip?: React.ReactNode;
};

const Input: React.FC<Props> = ({
  tooltip,
  type = "text",
  variant = "outlined",
  autoComplete = "off",
  InputProps,
  ...props
}): React.ReactElement => {
  return (
    <TextField
      type={type}
      variant={variant}
      autoComplete={autoComplete}
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
