import type { Account } from "../types.d";
import type { SelectChangeEvent } from "@mui/material";
import React from "react";
import Box from "@mui/material/Box";
import Form from "~/components/FormV2";

interface Props {
  onAccountChange: (login: string) => void;
  accounts: Account[];
  selected?: string;
  loading?: boolean;
}

const Accounts: React.FC<Props> = ({
  onAccountChange,
  accounts,
  selected,
  loading,
}) => {
  return (
    <Form.Select
      name="accounts"
      fullWidth
      disabled={loading}
      value={selected || ""}
      aria-label="Account selector"
      onChange={(e: SelectChangeEvent<unknown>) => {
        onAccountChange(e.target.value as string);
      }}
    >
      {accounts?.map(account => (
        <Form.Option key={account.id} value={account.id}>
          <Box
            component="img"
            src={account.avatar}
            alt={account.login}
            sx={{
              m: 0,
              width: 24,
              mr: 1.5,
              borderRadius: "50%",
              display: "inline-block",
            }}
          />
          {account.login}
        </Form.Option>
      ))}
    </Form.Select>
  );
};

export default Accounts;
