import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControlLabel from "@mui/material/FormControlLabel";
import Switch from "@mui/material/Switch";

interface Props {
  name: string;
  label: string;
  checked: boolean;
  setChecked: (val: boolean) => void;
  description: React.ReactNode;
}

export default function SwitchForm({
  name,
  label,
  checked,
  setChecked,
  description,
}: Props) {
  return (
    <Box sx={{ bgcolor: "container.paper", p: 1.75, pt: 1, mb: 4 }}>
      <FormControlLabel
        sx={{ pl: 0, ml: 0 }}
        label={label}
        control={
          <Switch
            name={name}
            color="secondary"
            checked={checked}
            onChange={e => {
              setChecked(e.target.checked);
            }}
          />
        }
        labelPlacement="start"
      />
      <Typography component="div" color="text.secondary">
        {description}
      </Typography>
    </Box>
  );
}
