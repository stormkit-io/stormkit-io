import type { SelectProps } from "@mui/material/Select";
import React, { useEffect, useMemo, useState } from "react";
import FilledInput from "@mui/material/FilledInput";
import OutlinedInput from "@mui/material/OutlinedInput";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import ListItemText from "@mui/material/ListItemText";
import ListSubheader from "@mui/material/ListSubheader";
import InputAdornment from "@mui/material/InputAdornment";
import Select from "@mui/material/Select";
import Checkbox from "@mui/material/Checkbox";
import FormHelperText from "@mui/material/FormHelperText";
import TextField from "@mui/material/TextField";
import SearchIcon from "@mui/icons-material/Search";

interface MenuItem {
  value: string;
  text: string;
}

interface Props extends Omit<SelectProps, "onSelect"> {
  items: MenuItem[];
  selected?: string[];
  label?: string;
  helperText?: React.ReactNode;
  emptyText?: string;
  placeholder?: string;
  onSearch?: (v: string) => void;
  onSelect: (v: string[]) => void;
}

export default function MultiSelect({
  items,
  helperText,
  selected,
  variant = "filled",
  size,
  label,
  placeholder,
  emptyText = "No item found",
  multiple = true,
  fullWidth = true,
  disabled = false,
  sx,
  onSearch,
  onSelect,
}: Props) {
  const [selectedItems, setSelectedItems] = useState<string[]>(selected || []);
  const [search, setSearch] = useState<string>();
  const filteredItems = useMemo(() => {
    if (!search?.length) {
      return items;
    }

    return items.filter(i => i.text.toLowerCase().includes(search));
  }, [search, items]);

  useEffect(() => {
    if (selected) {
      setSelectedItems(selected);
    }
  }, [selected]);

  return (
    <FormControl variant="standard" fullWidth={fullWidth}>
      {label && (
        <InputLabel id="multiple-checkbox-label" sx={{ pl: 2, pt: 1 }} shrink>
          {label}
        </InputLabel>
      )}
      <Select
        labelId="multiple-checkbox-label"
        multiple={multiple}
        variant={variant}
        size={size}
        disabled={disabled}
        value={selectedItems?.length ? selectedItems : [""]}
        input={variant === "filled" ? <FilledInput /> : <OutlinedInput />}
        MenuProps={{ autoFocus: false }}
        sx={sx}
        onClose={() => {
          // Rely on `onClose` only when `multiple` property is true.
          if (multiple) {
            onSelect(selectedItems.filter(i => i));
          }
        }}
        onChange={e => {
          let values = [
            ...new Set(
              typeof e.target.value === "string"
                ? e.target.value.split(",")
                : e.target.value
            ),
          ];

          const lastClicked = values.at(-1);

          if (typeof lastClicked === "undefined") {
            values = [];
          }

          setSelectedItems(values);

          // If not multiple, trigger the select here.
          if (!multiple) {
            onSelect(values.filter(i => i));
          }
        }}
        renderValue={renderValue => {
          // Find the `.text` property from items
          // If not exists, check if selected is provided and use the value
          // Otherwise use placeholder
          return (
            items
              .filter(s => renderValue.includes(s.value))
              .map(i => i.text)
              .join(", ") ||
            selected?.join(", ") ||
            placeholder
          );
        }}
      >
        <ListSubheader
          sx={{
            bgcolor: "transparent",
            p: 0,
            minWidth: 300,
            maxWidth: "calc(100vw - 32px)",
          }}
        >
          {onSearch && (
            <TextField
              variant="filled"
              placeholder="Search"
              autoFocus
              value={search || ""}
              fullWidth
              sx={{ mb: 1 }}
              autoComplete="off"
              data-testid="multiselect-search"
              slotProps={{
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                },
              }}
              onKeyDown={e => {
                if (e.key !== "Escape") {
                  // Prevents autoselecting item while typing (default Select behaviour)
                  e.stopPropagation();
                }
              }}
              onChange={e => {
                setSearch(e.target.value);
                onSearch(e.target.value);
              }}
            />
          )}
        </ListSubheader>
        {multiple ? (
          <MenuItem>
            <Checkbox checked={!selectedItems.filter(i => i).length} />
            <ListItemText primary={placeholder} />
          </MenuItem>
        ) : (
          // Fixes an issue with MUI Warning
          <MenuItem value={""} sx={{ display: "none" }} />
        )}
        {filteredItems.length ? (
          filteredItems.map(item => (
            <MenuItem key={item.value} value={item.value}>
              {multiple && (
                <Checkbox checked={selectedItems.includes(item.value)} />
              )}
              <ListItemText primary={item.text} />
            </MenuItem>
          ))
        ) : (
          <MenuItem>
            <ListItemText primary={emptyText} />
          </MenuItem>
        )}
      </Select>
      {helperText && <FormHelperText>{helperText}</FormHelperText>}
    </FormControl>
  );
}
