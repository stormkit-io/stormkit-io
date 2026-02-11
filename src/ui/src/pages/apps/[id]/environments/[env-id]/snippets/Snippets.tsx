import { useState, useContext } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import FormControlLabel from "@mui/material/FormControlLabel";
import Switch from "@mui/material/Switch";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CardRow from "~/components/CardRow";
import EmptyPage from "~/components/EmptyPage";
import ConfirmModal from "~/components/ConfirmModal";
import DomainSelector from "~/shared/domains/DomainSelector";
import { useFetchSnippets, deleteSnippet, updateSnippet } from "./actions";
import SnippetModal from "./SnippetModal";

interface SnippetLocationProps {
  prepend: boolean;
  location: "head" | "body" | undefined;
}

const SnippetLocation = ({ prepend, location }: SnippetLocationProps) => {
  const tag = location || "head";
  let text: string;

  if (prepend) {
    text = "Inserted right after <";
  } else {
    text = "Inserted right before </";
  }

  return (
    <Typography data-testid="snippet-location" sx={{ color: "text.secondary" }}>
      {text}
      <Typography
        component="span"
        sx={{ fontFamily: "monospace", fontSize: 11 }}
      >
        {tag}
        {">"}
      </Typography>
    </Typography>
  );
};

export default function Snippets() {
  const { app } = useContext(AppContext);
  const { environment: env } = useContext(EnvironmentContext);
  const [refreshToken, setRefreshToken] = useState(0);
  const [loadMoreToken, setLoadMoreToken] = useState<number>();
  const [isSnippetModalOpen, setIsSnippetModalOpen] = useState(false);
  const [toBeDeleted, setToBeDeleted] = useState<Snippet | undefined>();
  const [toBeToggled, setToBeToggled] = useState<Snippet | undefined>();
  const [toBeModified, setToBeModified] = useState<Snippet | undefined>();
  const [hosts, setHosts] = useState<string>();
  const { loading, error, snippets, hasNextPage, paymentRequired } =
    useFetchSnippets({
      app,
      env,
      refreshToken,
      loadMoreToken,
      hosts,
    });

  const hasFilters = Boolean(hosts);

  return (
    <Card error={error} loading={loading} sx={{ width: "100%" }}>
      <CardHeader
        title="Snippets"
        subtitle={
          <>
            Snippets will be injected during response time into your document.
            <br />
            You can enable or disable any snippet without the need of a
            deployment.
          </>
        }
      />
      {(hasFilters || snippets?.length) && (
        <Box sx={{ mb: 4, px: snippets?.length ? 4 : 0 }}>
          <DomainSelector
            variant="outlined"
            appId={app.id}
            envId={env.id!}
            fullWidth={false}
            onDomainSelect={d => {
              setLoadMoreToken(undefined);
              setHosts(d?.join(","));
            }}
            withDevDomains
            multiple
          />
        </Box>
      )}
      <Box>
        {snippets?.map(snippet => (
          <CardRow
            key={snippet.id}
            menuItems={[
              {
                icon: "fa fa-pencil",
                text: "Modify",
                onClick: () => {
                  setToBeDeleted(undefined);
                  setToBeToggled(undefined);
                  setToBeModified(snippet);
                  setIsSnippetModalOpen(true);
                },
              },
              {
                icon: "fa fa-times",
                text: "Delete",
                onClick: () => {
                  setToBeModified(undefined);
                  setToBeToggled(undefined);
                  setToBeDeleted(snippet);
                },
              },
            ]}
          >
            <Box sx={{ display: "flex" }}>
              <Box sx={{ flex: 1 }}>
                <Typography>{snippet.title}</Typography>
                <SnippetLocation
                  prepend={snippet.prepend}
                  location={snippet.location}
                />
                <Typography sx={{ color: "text.secondary" }}>
                  {snippet.rules?.hosts?.join(", ") || "All hosts"}
                </Typography>
              </Box>
              <FormControlLabel
                sx={{ pl: 0, ml: 0 }}
                label="Enabled"
                control={
                  <Switch
                    sx={{ mr: 2 }}
                    name="autoPublish"
                    color="secondary"
                    checked={snippet.enabled}
                    onChange={() => setToBeToggled(snippet)}
                  />
                }
                labelPlacement="start"
              />
            </Box>
          </CardRow>
        ))}
      </Box>
      {!snippets?.length && (
        <EmptyPage paymentRequired={paymentRequired}>
          {hasFilters ? (
            "Your search produced no results."
          ) : (
            <>
              It's quite empty in here.
              <br />
              Create a new snippet to manage them.
            </>
          )}
        </EmptyPage>
      )}
      <CardFooter sx={{ display: "flex", justifyContent: "center" }}>
        {hasNextPage && (
          <Button
            variant="text"
            color="info"
            type="button"
            sx={{ mr: 2 }}
            onClick={() => {
              setLoadMoreToken(Date.now());
            }}
          >
            Load more
          </Button>
        )}
        {!paymentRequired && !loading && (
          <Button
            variant="contained"
            color="secondary"
            type="button"
            onClick={() => {
              setIsSnippetModalOpen(true);
            }}
          >
            New Snippet
          </Button>
        )}
      </CardFooter>
      {isSnippetModalOpen && (
        <SnippetModal
          snippet={toBeModified}
          setRefreshToken={(d: number) => {
            setRefreshToken(d);
            setLoadMoreToken(undefined);
          }}
          closeModal={() => {
            setToBeModified(undefined);
            setIsSnippetModalOpen(false);
          }}
        />
      )}
      {toBeToggled && snippets && (
        <ConfirmModal
          onCancel={() => {
            setToBeToggled(undefined);
          }}
          onConfirm={({ setError, setLoading }) => {
            setLoading(true);
            updateSnippet({
              appId: app.id,
              envId: env.id!,
              snippet: {
                ...toBeToggled,
                enabled: !toBeToggled.enabled,
              },
            })
              .then(() => {
                setRefreshToken(Date.now());
                setToBeToggled(undefined);
              })
              .catch(res => {
                if (typeof res === "string") {
                  setError(res);
                } else {
                  setError("Something went wrong while updating snippet.");
                }
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          <p>
            This will{" "}
            <span className="font-bold">
              {toBeToggled.enabled ? "disable" : "enable"}
            </span>{" "}
            the snippet. Changes will be effective immediately.
          </p>
        </ConfirmModal>
      )}
      {toBeDeleted && snippets && (
        <ConfirmModal
          onCancel={() => {
            setToBeDeleted(undefined);
          }}
          onConfirm={({ setError, setLoading }) => {
            setLoading(true);
            deleteSnippet({
              snippet: toBeDeleted,
              appId: app.id,
              envId: env.id!,
            })
              .then(() => {
                setRefreshToken(Date.now());
                setToBeDeleted(undefined);
              })
              .catch(res => {
                if (typeof res === "string") {
                  setError(res);
                } else {
                  setError("Something went wrong while deleting snippet.");
                }
              })
              .finally(() => {
                setLoading(false);
              });
          }}
        >
          This will delete{" "}
          <Typography component="span" color="text.secondary" fontSize="medium">
            {toBeDeleted.title}
          </Typography>{" "}
          snippet and it won't be injected anymore.
        </ConfirmModal>
      )}
    </Card>
  );
}
