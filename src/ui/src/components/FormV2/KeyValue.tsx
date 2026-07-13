import React, { useEffect, useMemo, useState } from "react";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableFooter from "@mui/material/TableFooter";
import TableRow from "@mui/material/TableRow";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import AddIcon from "@mui/icons-material/Add";
import Button from "@mui/material/Button";
import TextFieldModal from "./KeyValueTextFieldModal";
import KeyValueRow from "./KeyValueRow";

interface Props {
  inputName: string;
  keyName: string;
  valName: string;
  keyIcons?: React.ReactNode | React.ReactNode[];
  keyPlaceholder?: string;
  valPlaceholder?: string;
  defaultValue: Record<string, string>;
  tdClasses?: string;
  thClasses?: string;
  resetToken?: number;
  separator?: string;
  isSensitive?: boolean;
  modifyAsString?: boolean;
  // When true, editing affordances (Add Row, Modify as a string) are disabled.
  disabled?: boolean;
  onChange?: (kv: Record<string, string>) => void;
  onModalOpen?: () => void;
}

const rowsToMap = (rows: string[][]): Record<string, string> => {
  return rows
    .filter(row => !row[2])
    .reduce((obj, val) => {
      if (typeof obj[val[0]] === "undefined") {
        obj[val[0]] = val[1];
      }

      return obj;
    }, {} as Record<string, string>);
};

export default function KeyValue({
  inputName,
  keyName,
  valName,
  keyIcons,
  keyPlaceholder,
  valPlaceholder,
  defaultValue,
  resetToken,
  separator = "=",
  isSensitive = false,
  modifyAsString = true,
  disabled = false,
  onChange,
  onModalOpen,
}: Props) {
  const [rows, setRows] = useState<string[][]>([]);
  const [isChanged, setIsChanged] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // We need to keep a reference to deleted rows because react lists get confused
  // when `key` might be the same. In our case, we can't provide anything for the `key` value
  // therefore it fallbacks to `index`.
  const rowsWithoutDeleted = useMemo(() => {
    return rows.filter(row => !row[2]);
  }, [rows]);

  // Sync with props
  useEffect(() => {
    const newRows: string[][] = [];

    Object.keys(defaultValue).forEach(key => {
      newRows.push([key, defaultValue[key]]);
    });

    setRows(newRows.length > 0 ? newRows : [["", ""]]);
    setIsChanged(false); // reset this value
  }, [defaultValue, resetToken]);

  // Let the parent know when rows change
  useEffect(() => {
    if (onChange && isChanged) {
      onChange(rowsToMap(rowsWithoutDeleted));
    }
  }, [rowsWithoutDeleted, isChanged]);

  return (
    <>
      <Table>
        <TableBody>
          {rows.map(([key, value, isDeleted], index) =>
            isDeleted ? undefined : (
              <KeyValueRow
                key={index}
                rows={rows}
                keyIcon={Array.isArray(keyIcons) ? keyIcons[index] : keyIcons}
                labelKey={keyName}
                labelValue={valName}
                inputKey={key}
                inputValue={value}
                inputName={inputName}
                index={index}
                isSensitive={isSensitive}
                hideByDefault={Boolean(value && isSensitive)}
                keyPlaceholder={keyPlaceholder}
                valPlaceholder={valPlaceholder}
                disabled={disabled}
                setIsChanged={setIsChanged}
                setRows={setRows}
              />
            )
          )}
        </TableBody>
        <TableFooter>
          <TableRow>
            <TableCell
              sx={{ borderBottom: "none", textAlign: "right", pr: 0 }}
              colSpan={2}
            >
              <Button
                color="primary"
                variant="outlined"
                type="button"
                disabled={disabled}
                sx={{
                  display: "inline-flex",
                  textTransform: "none",
                  mr: modifyAsString ? 2 : 0,
                }}
                onClick={e => {
                  e.preventDefault();
                  setRows([...rows, ["", ""]]);
                }}
              >
                <AddIcon sx={{ mr: 1, fontSize: 16 }} />
                Add Row
              </Button>

              {modifyAsString && (
                <Button
                  color="primary"
                  type="button"
                  variant="outlined"
                  disabled={disabled}
                  sx={{ display: "inline-flex", textTransform: "none" }}
                  onClick={() => {
                    setIsModalOpen(true);
                    onModalOpen?.();
                  }}
                >
                  <ContentCopyIcon sx={{ mr: 1, fontSize: 16 }} />
                  Modify as a string
                </Button>
              )}
            </TableCell>
          </TableRow>
        </TableFooter>
      </Table>
      {isModalOpen && (
        <TextFieldModal
          rows={rowsWithoutDeleted}
          separator={separator}
          isSensitive={isSensitive}
          placeholder={
            keyPlaceholder && valPlaceholder
              ? `${keyPlaceholder}${separator}${valPlaceholder}`
              : ""
          }
          onClose={() => {
            setIsModalOpen(false);
          }}
          onSave={rows => {
            setRows(
              rows
                .split("\n")
                .filter(i => i)
                .map(row => {
                  const indexOfEqual = row.indexOf(separator);

                  return [
                    row.slice(0, indexOfEqual),
                    row.slice(indexOfEqual + 1),
                  ];
                })
            );

            setIsChanged(true);
          }}
        />
      )}
    </>
  );
}
