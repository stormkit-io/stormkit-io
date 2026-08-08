import { useMemo, useState } from "react";
import MultiSelect from "~/components/MultiSelect";
import { debounce } from "@mui/material/utils";
import { useFetchDomains } from "./actions";

interface Props {
  envId: string;
  appId: string;
  multiple?: boolean;
  selected?: string[];
  variant?: "outlined" | "filled";
  size?: "small" | "medium";
  label?: string;
  fullWidth?: boolean;
  withDevDomains?: boolean;
  // Drops domains that are excluded from analytics. Only the analytics views
  // want this — snippets and triggers still target those hosts, which are
  // served normally.
  hideAnalyticsExcluded?: boolean;
  placeholder?: string;
  onFetch?: (d: Domain[]) => void;
  // If withDevDomains is true, returns selected domain names
  // Otherwise, returns the domains.
  onDomainSelect: (d: Domain[] | string[] | null) => void;
}

export default function DomainSelector({
  appId,
  envId,
  fullWidth,
  multiple = false,
  withDevDomains = false,
  hideAnalyticsExcluded = false,
  selected,
  label,
  size = "small",
  variant = "outlined",
  placeholder = "All domains",
  onFetch,
  onDomainSelect,
}: Props) {
  const [search, setSearch] = useState("");

  const visible = (d: Domain[]) =>
    hideAnalyticsExcluded ? d.filter(i => !i.analyticsExcluded) : d;

  const { domains, error } = useFetchDomains({
    appId,
    envId,
    verified: true,
    search,
    onFetch: onFetch && (d => onFetch(visible(d))),
  });

  const selectable = visible(domains || []);

  const items = useMemo(() => {
    return [
      withDevDomains
        ? { value: "*.dev", text: "All development endpoints (*.dev)" }
        : { value: "", text: "" },
      ...selectable.map(d => ({ value: d.domainName, text: d.domainName })),
    ].filter(i => i.value);
  }, [withDevDomains, selectable]);

  if (error) {
    return <></>;
  }

  return (
    <MultiSelect
      emptyText="No domain found"
      label={label}
      variant={variant}
      size={size}
      placeholder={placeholder}
      fullWidth={fullWidth}
      multiple={multiple}
      items={items}
      selected={selected}
      onSearch={debounce(setSearch, 300)}
      onSelect={values => {
        if (withDevDomains) {
          onDomainSelect(values);
        } else {
          onDomainSelect(selectable.filter(d => values.includes(d.domainName)));
        }
      }}
    />
  );
}
