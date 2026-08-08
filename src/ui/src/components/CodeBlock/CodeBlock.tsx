import type { SxProps } from "@mui/material";
import { useMemo } from "react";
import Box from "@mui/material/Box";
import {
  detectLanguage,
  highlightToHtml,
  tokenStyles,
  type Language,
} from "~/utils/helpers/highlight";

interface Props {
  children: string;
  /** Defaults to detecting json/html from the content. */
  language?: Language;
  sx?: SxProps;
}

const preStyles = {
  bgcolor: "container.transparent",
  p: 2,
  maxWidth: "100%",
  overflow: "auto",
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
};

// Detecting and highlighting both parse the whole input synchronously during
// render, and expand it into one element per token. Trigger log bodies are
// whatever the remote endpoint returned, with no cap, so above this size the
// block renders as plain text — the same way it did before it was highlighted.
const MAX_HIGHLIGHT_LENGTH = 100_000;

/**
 * Renders a block of code, syntax highlighted when the language is one we can
 * parse and as plain text otherwise.
 */
export default function CodeBlock({ children, language, sx }: Props) {
  const html = useMemo(() => {
    if (children.length > MAX_HIGHLIGHT_LENGTH) {
      return null;
    }

    return highlightToHtml(children, language ?? detectLanguage(children));
  }, [children, language]);

  const styles = { ...preStyles, ...tokenStyles, ...sx };

  if (html === null) {
    return (
      <Box component="pre" data-testid="code-block" sx={styles}>
        {children}
      </Box>
    );
  }

  return (
    <Box
      component="pre"
      data-testid="code-block"
      sx={styles}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
