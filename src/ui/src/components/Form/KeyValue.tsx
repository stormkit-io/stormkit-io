import React, { useEffect, useState } from "react";
import { TextFieldProps } from "@mui/material/TextField";
import Input from "./Input";
import Button from "~/components/Button";

/**
 * This function generates alternative keys for the values array
 * in order to help React remember the form state whenever the rows
 * change. Each time a new row is added, Date.now() is assigned to
 * that rows altKey property. If this property misses - first render for
 * instance - then the array index is assigned.
 */
const generateAltKeys = (vals: Array<KeyValue>): Array<KeyValue> => {
  return vals.map((value, i) => ({
    ...value,
    altKey: value.altKey ?? i,
  }));
};

interface KeyValue {
  key: string;
  value: string;
  altKey?: number;
}

interface Props {
  keyProps: TextFieldProps;
  valueProps: TextFieldProps;
  defaultValues?: Array<KeyValue>;
  plusButtonAriaLabel?: string;
  onChange: (val: Array<KeyValue>) => void;
}

const KeyValue: React.FC<Props> = ({
  keyProps,
  valueProps,
  defaultValues = [],
  plusButtonAriaLabel,
  onChange,
}): React.ReactElement => {
  const [values, setValues] = useState<Array<KeyValue>>(
    generateAltKeys(defaultValues)
  );

  useEffect(() => {
    if (defaultValues?.length === 0) {
      setValues([{ key: "", value: "", altKey: Date.now() }]);
    } else {
      setValues(generateAltKeys(defaultValues));
    }
  }, [defaultValues]);

  return (
    <>
      {values.map(({ key, value, altKey }, i) => (
        <div className="flex justify-between mb-2 relative" key={altKey}>
          <div className="flex flex-auto items-center mr-2">
            {i === values.length - 1 && (
              <Button
                styled={false}
                className="absolute left-0 -ml-6"
                type="button"
                aria-label={plusButtonAriaLabel}
                onClick={() => {
                  onChange([
                    ...values,
                    { key: "", value: "", altKey: Date.now() },
                  ]);
                }}
              >
                <span className="fas fa-plus-circle" />
              </Button>
            )}
            <Input
              multiline
              defaultValue={key}
              fullWidth
              onBlur={e => {
                const copy = values.slice(0);
                copy[i].key = e.target.value;
                onChange(copy);
              }}
              {...keyProps}
            />
          </div>
          <div className="flex flex-auto items-center max-w-1/2 relative">
            <Input
              defaultValue={value}
              fullWidth
              onBlur={e => {
                const copy = values.slice(0);
                copy[i].value = e.target.value;
                onChange(copy);
              }}
              {...valueProps}
            />

            <Button
              styled={false}
              className="absolute right-0 -mr-6"
              type="button"
              onClick={() => {
                const copy = values.slice(0);
                copy.splice(i, 1);
                onChange(copy);
              }}
            >
              <span className="fas fa-minus-circle red-50" />
            </Button>
          </div>
        </div>
      ))}
    </>
  );
};

export default KeyValue;
