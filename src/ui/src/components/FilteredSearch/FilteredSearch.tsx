import { useEffect, useMemo, useRef, useState } from "react";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Popper from "@mui/material/Popper";
import MenuList from "@mui/material/MenuList";
import MenuItem from "@mui/material/MenuItem";
import ListSubheader from "@mui/material/ListSubheader";
import InputBase from "@mui/material/InputBase";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import CircularProgress from "@mui/material/CircularProgress";
import ClickAwayListener from "@mui/material/ClickAwayListener";
import SearchIcon from "@mui/icons-material/Search";
import type { FilterDef, FilterValues } from "./types";
import { nowPreset, relativePresets, toInputValue } from "./datetime";
import Token from "./Token";

interface Suggestion {
  text: string;
  /** Deferred so relative dates resolve when picked, not when rendered. */
  resolve: () => string;
}

interface Props {
  defs: FilterDef[];
  values: FilterValues;
  /**
   * `replace` marks a change that undoes rather than narrows the search, so a
   * consumer persisting to history does not make Back resurrect filters the
   * user just cleared.
   */
  onChange: (values: FilterValues, options: { replace: boolean }) => void;
  placeholder?: string;
}

const matches = (haystack: string, needle: string) =>
  haystack.toLowerCase().includes(needle.trim().toLowerCase());

export default function FilteredSearch({
  defs,
  values,
  onChange,
  placeholder = "Filter or search",
}: Props) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const [pendingKey, setPendingKey] = useState<string>();
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  /** Whether the highlight was moved on purpose, rather than defaulted to 0. */
  const [arrowed, setArrowed] = useState(false);
  const [remote, setRemote] = useState<Suggestion[]>([]);
  const [remoteLoading, setRemoteLoading] = useState(false);

  const pendingDef = defs.find(d => d.key === pendingKey);
  const tokens = defs.filter(d => values[d.key]);

  useEffect(() => {
    if (!pendingDef?.search) {
      setRemote([]);
      setRemoteLoading(false);
      return;
    }

    let unmounted = false;
    setRemoteLoading(true);

    const timer = setTimeout(() => {
      pendingDef.search!(text)
        .then(options => {
          if (!unmounted) {
            setRemote(
              options.map(o => ({ text: o.text, resolve: () => o.value })),
            );
          }
        })
        .catch(() => {
          if (!unmounted) {
            setRemote([]);
          }
        })
        .finally(() => {
          if (!unmounted) {
            setRemoteLoading(false);
          }
        });
    }, 250);

    return () => {
      unmounted = true;
      clearTimeout(timer);
    };
  }, [pendingKey, text]);

  const suggestions = useMemo<Suggestion[]>(() => {
    if (!pendingDef) {
      return defs
        .filter(d => !values[d.key] && matches(d.label, text))
        .map(d => ({ text: d.label, resolve: () => d.key }));
    }

    if (pendingDef.kind === "datetime") {
      return [nowPreset, ...relativePresets]
        .filter(p => matches(p.text, text))
        .map(p => ({ text: p.text, resolve: p.value }));
    }

    if (pendingDef.search) {
      return remote;
    }

    return (pendingDef.options || [])
      .filter(o => matches(o.text, text))
      .map(o => ({ text: o.text, resolve: () => o.value }));
  }, [pendingDef, defs, values, text, remote]);

  const commit = (raw: string) => {
    if (!pendingDef) {
      return;
    }

    const value = (
      pendingDef.normalize ? pendingDef.normalize(raw) : raw
    ).trim();

    if (!value || (pendingDef.validate && !pendingDef.validate(value))) {
      return;
    }

    onChange({ ...values, [pendingDef.key]: value }, { replace: false });
    setPendingKey(undefined);
    setText("");
    setActiveIndex(0);
    setArrowed(false);
  };

  const select = (index: number) => {
    const suggestion = suggestions[index];

    if (!suggestion) {
      return;
    }

    if (!pendingDef) {
      setPendingKey(suggestion.resolve());
      setText("");
      setActiveIndex(0);
      setArrowed(false);
      return;
    }

    commit(suggestion.resolve());
  };

  const remove = (key: string) => {
    const next = { ...values };
    delete next[key];
    onChange(next, { replace: true });
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex(i => (i + 1) % Math.max(suggestions.length, 1));
      setArrowed(true);
      setOpen(true);
      return;
    }

    if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex(
        i => (i - 1 + suggestions.length) % Math.max(suggestions.length, 1),
      );
      setArrowed(true);
      return;
    }

    if (e.key === "Enter") {
      e.preventDefault();

      // A typed value wins over the merely-defaulted highlight, otherwise a
      // free-text filter could never be committed while suggestions are shown.
      // Moving the highlight on purpose opts back into picking it.
      if (pendingDef && text.trim() && !(arrowed && suggestions[activeIndex])) {
        commit(text);
        return;
      }

      select(activeIndex);
      return;
    }

    if (e.key === "Escape") {
      setOpen(false);
      return;
    }

    if (e.key === "Backspace" && !text) {
      if (pendingDef) {
        setPendingKey(undefined);
        return;
      }

      const last = tokens.at(-1);

      if (last) {
        remove(last.key);
      }
    }
  };

  const showCustomDate = pendingDef?.kind === "datetime";
  const showFreeTextHint =
    pendingDef && !showCustomDate && !pendingDef.options && !pendingDef.search;

  return (
    <ClickAwayListener onClickAway={() => setOpen(false)}>
      <Box sx={{ width: "100%" }}>
        <Box
          ref={setAnchorEl}
          data-testid="filtered-search"
          onClick={() => {
            setOpen(true);
            inputRef.current?.focus();
          }}
          sx={{
            display: "flex",
            alignItems: "center",
            flexWrap: "wrap",
            gap: 1,
            px: 2,
            py: 1,
            cursor: "text",
            borderRadius: 1,
            bgcolor: "container.paper",
            border: "1px solid",
            borderColor: open ? "primary.main" : "container.border",
          }}
        >
          <SearchIcon sx={{ opacity: 0.5, fontSize: 20 }} />
          {tokens.map(def => (
            <Token
              key={def.key}
              testId={`token-${def.key}`}
              label={def.label}
              value={def.format ? def.format(values[def.key]) : values[def.key]}
              onDelete={() => remove(def.key)}
            />
          ))}
          {pendingDef && (
            <Token
              pending
              testId="token-pending"
              label={pendingDef.label}
              onDelete={() => setPendingKey(undefined)}
            />
          )}
          <InputBase
            inputRef={inputRef}
            value={text}
            sx={{ flex: 1, minWidth: 160 }}
            inputProps={{
              "aria-label": placeholder,
              type: pendingDef?.kind === "number" ? "number" : "text",
            }}
            placeholder={
              pendingDef ? `Value for ${pendingDef.label}` : placeholder
            }
            onFocus={() => setOpen(true)}
            onKeyDown={onKeyDown}
            onChange={e => {
              setText(e.target.value);
              setActiveIndex(0);
              setArrowed(false);
              setOpen(true);
            }}
          />
          {tokens.length > 0 && (
            <Typography
              component="button"
              type="button"
              variant="caption"
              data-testid="clear-all"
              sx={{
                flexShrink: 0,
                border: 0,
                p: 0,
                bgcolor: "transparent",
                color: "text.primary",
                cursor: "pointer",
                opacity: 0.7,
                textDecoration: "underline",
                ":hover": { opacity: 1 },
              }}
              onClick={e => {
                e.stopPropagation();
                setPendingKey(undefined);
                setText("");
                onChange({}, { replace: true });
              }}
            >
              Clear all
            </Typography>
          )}
        </Box>
        <Popper
          open={open}
          anchorEl={anchorEl}
          placement="bottom-start"
          sx={{ zIndex: 1300, width: anchorEl?.clientWidth }}
        >
          <Paper sx={{ mt: 0.5, maxHeight: 320, overflowY: "auto" }}>
            {pendingDef?.searchHint && (
              <ListSubheader sx={{ bgcolor: "transparent", lineHeight: 2.5 }}>
                {pendingDef.searchHint}
              </ListSubheader>
            )}
            {remoteLoading && (
              <Box sx={{ display: "flex", justifyContent: "center", py: 2 }}>
                <CircularProgress size={16} />
              </Box>
            )}
            {!remoteLoading && suggestions.length > 0 && (
              <MenuList dense>
                {suggestions.map((s, i) => (
                  <MenuItem
                    key={s.text}
                    selected={i === activeIndex}
                    onMouseEnter={() => setActiveIndex(i)}
                    onClick={() => select(i)}
                  >
                    {s.text}
                  </MenuItem>
                ))}
              </MenuList>
            )}
            {!remoteLoading && suggestions.length === 0 && !showCustomDate && (
              <Typography sx={{ px: 2, py: 1.5, opacity: 0.6 }} variant="body2">
                {showFreeTextHint
                  ? "Type a value and press Enter"
                  : "No matching filter"}
              </Typography>
            )}
            {showCustomDate && (
              <Box sx={{ px: 2, py: 1.5 }}>
                <TextField
                  variant="filled"
                  size="small"
                  fullWidth
                  type="datetime-local"
                  label="Custom"
                  data-testid="custom-datetime"
                  defaultValue={toInputValue(
                    pendingDef ? values[pendingDef.key] || "" : "",
                  )}
                  InputLabelProps={{ shrink: true }}
                  onChange={e => commit(e.target.value)}
                />
              </Box>
            )}
            {pendingDef && !showCustomDate && (
              <Typography
                variant="caption"
                sx={{ display: "block", px: 2, py: 1, opacity: 0.6 }}
              >
                Press Enter to use exactly what you typed
              </Typography>
            )}
          </Paper>
        </Popper>
      </Box>
    </ClickAwayListener>
  );
}
