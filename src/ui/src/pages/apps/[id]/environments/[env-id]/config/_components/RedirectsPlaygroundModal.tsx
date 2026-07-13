import { FormEventHandler, useState } from "react";
import Table from "@mui/material/Table";
import TableRow from "@mui/material/TableRow";
import TableCell from "@mui/material/TableCell";
import TableBody from "@mui/material/TableBody";
import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import Button from "@mui/material/Button";
import TextInput from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";
import { grey } from "@mui/material/colors";
import RedirectsEditor from "./Editor";

interface Props {
  env: Environment;
  appId: string;
  onClose: () => void;
}

interface SubmitReturn {
  match: boolean;
  against: string;
  pattern: string;
  rewrite: string;
  redirect: string;
  proxy: boolean;
  status: number;
}

const exampleRedirects = `[
    { "from": "/my-path", "to": "/my-new-path", "status": 302 }
]`;

export default function RedirectsPlaygroundModal({
  onClose,
  appId,
  env,
}: Props) {
  const [address, setAddress] = useState(env.preview);
  const [result, setResult] = useState<SubmitReturn>();
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [redirects, setRedirects] = useState(exampleRedirects);

  const submitHandler: FormEventHandler = e => {
    e.preventDefault();
    e.stopPropagation();

    let parsed: any;

    try {
      parsed = JSON.parse(redirects);
    } catch {
      setError("Invalid JSON format.");
      return;
    }

    if (!address.trim()) {
      setError("Request URL cannot be empty.");
      return;
    }

    setLoading(true);

    api
      .post<SubmitReturn>("/redirects/playground", {
        appId: appId,
        envId: env.id,
        address,
        redirects: parsed,
      })
      .then(match => {
        setResult(match);
      })
      .finally(() => {
        setError(undefined);
        setLoading(false);
      });
  };

  return (
    <Modal open onClose={onClose} maxWidth="lg">
      <Card
        data-testid="redirects-playground-form"
        component="form"
        onSubmit={submitHandler}
        error={
          error || !env.published
            ? error ||
              "Redirects are tested against the published deployments. Please deploy and publish before testing redirects."
            : undefined
        }
      >
        <CardHeader title="Redirects Playground" />
        <Box sx={{ mb: 4 }}>
          <TextInput
            variant="filled"
            label="Request URL"
            placeholder="e.g. https://www.stormkit.io/docs"
            fullWidth
            autoFocus
            autoComplete="off"
            value={address}
            onChange={e => setAddress(e.target.value)}
            helperText={
              <Typography sx={{ mt: 1 }} component="span">
                Provide a URL to test against the redirects.
              </Typography>
            }
          />
        </Box>
        <Box
          sx={{
            mb: 4,
            p: 2,
            bgcolor: "container.paper",
            borderBottom: `1px solid ${grey[900]}`,
          }}
        >
          <Typography sx={{ mb: 2, color: "text.secondary" }}>
            Redirects
          </Typography>
          <RedirectsEditor
            value={redirects}
            onChange={setRedirects}
            docsLink="https://www.stormkit.io/docs/features/redirects-and-path-rewrites"
          />
        </Box>
        {result &&
          (result.match ? (
            <>
              <Alert color="success" sx={{ mb: 0 }}>
                <Typography>It's a match!</Typography>
              </Alert>
              <Table sx={{ mb: 4 }}>
                <TableBody>
                  <TableRow>
                    <TableCell>Status</TableCell>
                    <TableCell>
                      {result.status
                        ? `Redirect ${result.status}`
                        : "Path Rewrite"}
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell>To</TableCell>
                    <TableCell>{result.redirect || result.rewrite}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell>Should proxy?</TableCell>
                    <TableCell>{result.proxy ? "Yes" : "No"}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </>
          ) : (
            <Alert color="error" sx={{ mb: 4 }}>
              <AlertTitle>Not a match</AlertTitle>
              <Typography>
                Provided domain did not match any redirect rule.
              </Typography>
            </Alert>
          ))}
        <CardFooter>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            disabled={!env.published}
            loading={loading}
          >
            Test
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
