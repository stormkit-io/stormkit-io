import { useContext } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { RootContext } from "~/pages/Root.context";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";

interface Props {
  value?: string;
  docsLink: string;
  onChange: (r: string) => void;
}

export default function Editor({ value, docsLink, onChange }: Props) {
  const { mode } = useContext(RootContext);

  return (
    <div style={{ minWidth: 0, overflow: "hidden" }}>
      <CodeMirror
        maxHeight="200px"
        value={value}
        extensions={[json()]}
        onChange={onChange}
        theme={mode === "dark" ? "dark" : "light"}
      />
      <Typography sx={{ mt: 2, color: "text.secondary" }}>
        Check the{" "}
        <Link href={docsLink} target="_blank" rel="noreferrer noopener">
          docs
        </Link>{" "}
        for more information.
      </Typography>
    </div>
  );
}
