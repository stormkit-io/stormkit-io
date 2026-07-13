import React, { useRef } from "react";
import FormControlLabel from "@mui/material/FormControlLabel";
import Input from "./Input";
import Checkbox from "./Checkbox";
import Radio from "./Radio";
import Select from "./Select";
import Option from "./Option";
import Toggler from "./Toggler";
import Switch from "./Switch";
import Helper from "./Helper";
import Header from "./Header";
import KeyValue from "./KeyValue";
import Section from "./components/Section";
import Description from "./components/Description";
import CodeMirror from "@uiw/react-codemirror";

type FormRef = React.RefObject<HTMLFormElement>;
type FormValue = boolean | string;

interface Props<T> extends React.HTMLProps<HTMLFormElement> {
  method?: "GET";
  autoComplete?: "off" | "on";
  ignoreDisabled?: boolean;
  ignoreEmptyValues?: boolean;
  handleSubmit?: (value: T) => void;
}

const getElementValue = (el: HTMLFormElement): FormValue => {
  const type = el.getAttribute("type");

  if (type === "checkbox") {
    return el.checked ? el.value : false;
  }

  return el.value;
};

function collectFormValues<T>(
  ref: FormRef,
  opts: { ignoreEmptyValues: boolean; ignoreDisabled: boolean }
): T {
  if (!ref || ref.current === null) {
    return {} as T;
  }

  const form = ref.current;
  const values: { [key: string]: FormValue | Array<FormValue> } = {};

  for (let i = 0; i < form.elements.length; i++) {
    const el = form.elements[i] as HTMLFormElement;
    const { name, disabled } = el;
    const value = getElementValue(el);

    // Do not continue if a name has not been specified.
    // Do not continue if ignoreEmptyValues is true and value is false-ish.
    // Do not continue if input is disabled and ignoreDisabled is true-ish.
    if (
      !name ||
      (opts.ignoreEmptyValues && !value) ||
      (disabled && opts.ignoreDisabled)
    ) {
      continue;
    }

    const type = el.getAttribute("type");

    if (type === "radio") {
      values[name] = el.checked ? value : values[name];
    } else if (typeof values[name] !== "undefined") {
      values[name] = Array.isArray(values[name])
        ? (values[name] as Array<FormValue>)
        : [values[name] as FormValue];

      (values[name] as FormValue[]).push(value);
    } else {
      values[name] = value;
    }
  }

  return values as unknown as T;
}

interface FormSubmitProps<T> extends Props<T> {
  ref: FormRef;
}

function useFormSubmit<T>({
  ref,
  handleSubmit,
  ignoreEmptyValues = false,
  ignoreDisabled = false,
}: FormSubmitProps<T>) {
  return (e: React.FormEvent) => {
    e.preventDefault();

    const values = collectFormValues<T>(ref, {
      ignoreEmptyValues,
      ignoreDisabled,
    });

    handleSubmit && handleSubmit(values);
  };
}

function Form<T>({
  children,
  handleSubmit,
  ignoreEmptyValues,
  ignoreDisabled,
  ...props
}: Props<T>): React.ReactElement {
  const ref = useRef<HTMLFormElement>(null);
  const handleFormSubmit = useFormSubmit<T>({
    ref,
    handleSubmit,
    ignoreDisabled,
    ignoreEmptyValues,
  });

  return (
    <form {...props} ref={ref} onSubmit={handleFormSubmit}>
      {children}
    </form>
  );
}

Form.Input = Input;
Form.ControlLabel = FormControlLabel;
Form.Code = CodeMirror;
Form.Checkbox = Checkbox;
Form.Radio = Radio;
Form.Select = Select;
Form.Option = Option;
Form.Toggler = Toggler;
Form.Switch = Switch;
Form.Section = Section;
Form.Description = Description;
Form.Helper = Helper;
Form.Header = Header;
Form.KeyValue = KeyValue;

export default Form;
