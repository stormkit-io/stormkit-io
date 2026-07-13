import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import Modal from "~/components/Modal";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import { usePersonalAccessTokenState as usePATState } from "../actions";
import CardFooter from "~/components/CardFooter";

interface Props {
  hasToken: boolean;
  toggleModal: (val: boolean) => void;
}

export default function PersonalAccessTokenModal({
  hasToken,
  toggleModal,
}: Props) {
  const state = usePATState({ hasToken });

  return (
    <Modal open onClose={() => toggleModal(false)}>
      <Card
        success={state.msg?.type === "success" && state.msg.content}
        error={state.msg?.type === "error" && state.msg.content}
        info={
          hasToken
            ? "There is already a personal access token associated with this account. Submit a new one to overwrite."
            : ""
        }
      >
        <CardHeader
          title={`${hasToken ? "Reset" : "Set"} personal access token`}
        />
        <TextField
          label="Token"
          aria-label="Personal access token"
          type="text"
          variant="filled"
          autoComplete="off"
          placeholder="Your personal access token that will be used for oauth"
          onChange={e => state.setToken(e.target.value)}
          fullWidth
          autoFocus
          sx={{ mb: 4 }}
        />
        <CardFooter sx={{ textAlign: "center" }}>
          {hasToken ? (
            <Button
              variant="text"
              type="button"
              onClick={state.deleteToken}
              loading={state.loading === "delete"}
              className="mr-4"
            >
              Delete existing token
            </Button>
          ) : (
            ""
          )}
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            onClick={state.submitToken}
            loading={state.loading === "submit"}
            disabled={!state.token}
          >
            Submit
          </Button>
        </CardFooter>
      </Card>
    </Modal>
  );
}
