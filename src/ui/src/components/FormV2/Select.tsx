import React from "react";
import cn from "classnames";
import InputLabel from "@mui/material/InputLabel";
import FormControl from "@mui/material/FormControl";
import SelectInput, { SelectProps } from "@mui/material/Select";

let inputLabel = 0;

interface Props {
  shrink?: boolean;
}

const Select: React.FC<SelectProps & Props> = ({
  className,
  variant = "filled",
  label,
  shrink = false,
  fullWidth = true,
  displayEmpty = true,
  defaultValue = "",
  required,
  ...rest
}): React.ReactElement => {
  const labelId = `form-select-${inputLabel++}`;

  return (
    <FormControl
      color="info"
      variant={variant}
      fullWidth={fullWidth}
      required={required}
      sx={{
        height: "100%",
      }}
    >
      {label && (
        <InputLabel id={labelId} shrink={shrink}>
          {label}
        </InputLabel>
      )}
      <SelectInput
        label={label}
        labelId={labelId}
        className={cn({ "w-full": fullWidth }, className)}
        displayEmpty={displayEmpty}
        defaultValue={defaultValue}
        inputProps={{
          sx: { lineHeight: 1.5, p: 2 },
        }}
        {...rest}
      />
    </FormControl>
  );
};

export default Select;
