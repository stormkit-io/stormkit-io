import type { FormEventHandler } from "react";
import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";

interface Props {
  appId: string;
  envId: string;
  defaultFrom: string;
  defaultTo: string;
  onClose: () => void;
}

export default function SendTestEmailModal({
  appId,
  envId,
  defaultFrom,
  defaultTo,
  onClose,
}: Props) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();

  const handleSubmit: FormEventHandler<HTMLElement> = e => {
    e.preventDefault();
    // The modal is portaled from within the parent configuration form;
    // without this the submit event bubbles up the React tree and
    // triggers the parent form's submit handler as well.
    e.stopPropagation();

    const form = e.target as HTMLFormElement;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<
      string,
      string
    >;

    if (data.from === "" || data.to === "") {
      return setError("From and To are required fields.");
    }

    setIsLoading(true);
    setError(undefined);
    setSuccess(undefined);

    api
      .post<{ ok: boolean; delivered: boolean }>("/v1/mail", {
        // fetch treats the `body` argument as a json string so we
        // need to stringify the parameters to make this api call work.
        body: JSON.stringify({
          appId,
          envId,
          from: data.from,
          to: data.to,
          body: "Test email body",
          subject: "Test email subject",
        }),
      })
      .then(({ delivered }) => {
        // Only an explicit false means undelivered. Treating a missing flag as
        // a failure would report a delivered email as unsent.
        if (delivered === false) {
          return setError(
            "This environment has no SMTP configuration, so the email was recorded but never sent. Save a mailer configuration first."
          );
        }

        setSuccess("Test email sent to " + data.to);
      })
      .catch(() => {
        setError("Something went wrong while sending test email.");
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <Modal open onClose={onClose}>
      <Card
        component="form"
        error={error}
        success={success}
        onSubmit={handleSubmit}
      >
        <CardHeader
          title="Send test email"
          subtitle="Verify your mailer configuration by sending a test email."
        />

        <Box sx={{ mb: 4 }}>
          <TextField
            label="From"
            name="from"
            fullWidth
            defaultValue={defaultFrom}
            variant="filled"
            autoComplete="off"
            placeholder="sender@example.com"
          />
        </Box>

        <Box sx={{ mb: 4 }}>
          <TextField
            label="To"
            name="to"
            fullWidth
            defaultValue={defaultTo}
            variant="filled"
            autoComplete="off"
            placeholder="recipient@example.com"
          />
        </Box>

        <CardFooter>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={isLoading}
          >
            Send
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
