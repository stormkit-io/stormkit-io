import { useContext } from "react";
import Link, { LinkProps } from "@mui/material/Link";
import { AuthContext } from "~/pages/auth/Auth.context";

interface Props extends LinkProps {
  repo: string;
  provider: Provider;
  sha?: string;
}

export default function Sha({ repo, provider, sha, sx, ...rest }: Props) {
  const { providers } = useContext(AuthContext);

  if (!sha) {
    return <></>;
  }

  const shaShort = sha.substring(0, 7);
  let link = "";

  if (provider == "github") {
    link = `https://${repo.replace("github", "github.com")}/commit/${shaShort}`;
  } else if (provider == "gitlab") {
    link = `https://${repo.replace("gitlab", "gitlab.com")}/-/commit/${shaShort}`;
  } else if (provider == "forgejo") {
    link = providers?.forgejoUrl
      ? `${repo.replace("forgejo", providers.forgejoUrl)}/commit/${shaShort}`
      : "";
  } else {
    link = `https://${repo.replace("bitbucket", "bitbucket.org")}/commits/${shaShort}`;
  }

  return (
    <Link
      href={link || undefined}
      sx={{ ...sx, fontSize: "inherit", color: "inherit" }}
      {...rest}
    >
      #{sha.substring(0, 6)}
    </Link>
  );
}
