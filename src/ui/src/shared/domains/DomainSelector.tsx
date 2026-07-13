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
  selected,
  label,
  size = "small",
  variant = "outlined",
  placeholder = "All domains",
  onFetch,
  onDomainSelect,
}: Props) {
  const [search, setSearch] = useState("");
  const { domains, error } = useFetchDomains({
    appId,
    envId,
    verified: true,
    search,
    onFetch,
  });

  const items = useMemo(() => {
    return [
      withDevDomains
        ? { value: "*.dev", text: "All development endpoints (*.dev)" }
        : { value: "", text: "" },
      ...domains?.map(d => ({ value: d.domainName, text: d.domainName })),
    ].filter(i => i.value);
  }, [withDevDomains, domains]);

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
          onDomainSelect(domains.filter(d => values.includes(d.domainName)));
        }
      }}
    />
  );
}
