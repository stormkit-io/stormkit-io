import Box from "@mui/material/Box";
import { json } from "@codemirror/lang-json";
import Button from "@mui/material/Button";
import CodeMirror from "@uiw/react-codemirror";
import Typography from "@mui/material/Typography";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";

interface Props {
  audit: Audit;
  onClose: () => void;
}

export default function DiffModal({ onClose, audit }: Props) {
  const oldDiff = audit.diff.old;
  const newDiff = audit.diff.new;
  const hasOldDiff = Object.keys(oldDiff).length > 0;
  const hasNewDiff = Object.keys(newDiff).length > 0;

  return (
    <Modal open maxWidth="lg" onClose={onClose}>
      <Card>
        <CardHeader
          title="Diff"
          subtitle="Visualize changes with this activity item"
        />
        <Box sx={{ display: "flex", mb: 4 }}>
          {hasOldDiff && (
            <Box
              sx={{
                flex: 1,
                mr: 2,
                maxWidth: hasNewDiff ? "50%" : undefined,
              }}
            >
              <Typography sx={{ mb: 2 }} variant="h4">
                Old version
              </Typography>
              <CodeMirror
                basicSetup={{ lineNumbers: false, foldGutter: false }}
                value={JSON.stringify(oldDiff, null, 2)}
                extensions={[json()]}
                readOnly
                theme="dark"
              />
            </Box>
          )}
          {hasNewDiff && (
            <Box
              sx={{
                flex: 1,
                maxWidth: hasOldDiff ? "50%" : undefined,
                width: "100%",
              }}
            >
              <Typography sx={{ mb: 2 }} variant="h4">
                New version
              </Typography>
              <CodeMirror
                basicSetup={{ lineNumbers: false, foldGutter: false }}
                value={JSON.stringify(newDiff, null, 2)}
                extensions={[json()]}
                readOnly
                theme="dark"
              />
            </Box>
          )}
        </Box>
        <CardFooter>
          <Button variant="contained" color="secondary" onClick={onClose}>
            Close
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
