import React from "react";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import Select from "@mui/material/Select";
import Option from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";

interface Props extends Omit<React.HTMLAttributes<HTMLDivElement>, "onSelect"> {
  environments: Array<Environment>;
  placeholder: string;
  onSelect: (arg0: Environment) => void;
  defaultValue?: string;
}

export default function EnvironmentSelector({
  environments,
  defaultValue = "",
  placeholder = "Select an environment",
  onSelect,
}: Props) {
  return (
    <FormControl variant="standard" fullWidth>
      <InputLabel id="request-method-label" sx={{ pl: 2, pt: 1 }}>
        {placeholder}
      </InputLabel>
      <Select
        name="envId"
        variant="filled"
        displayEmpty
        defaultValue={defaultValue}
        fullWidth
        onChange={e => {
          const id = e.target.value as string;
          onSelect(environments.filter(e => e.id === id)[0]);
        }}
      >
        {environments.map(env => (
          <Option value={env.id!} key={env.id}>
            <Typography component="span">
              {env.name || env.env}{" "}
              <Typography component="span" color="text.secondary">
                {env.preview}
              </Typography>
            </Typography>
          </Option>
        ))}
      </Select>
    </FormControl>
  );
}
