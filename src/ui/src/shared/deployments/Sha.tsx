import Link, { LinkProps } from "@mui/material/Link";

interface Props extends LinkProps {
  repo: string;
  provider: Provider;
  sha?: string;
}

export default function Sha({ repo, provider, sha, sx, ...rest }: Props) {
  if (!sha) {
    return <></>;
  }

  const shaShort = sha.substring(0, 7);
  let link = "";

  if (provider == "github") {
    link = `${repo.replace("github", "github.com")}/commit/${shaShort}`;
  } else if (provider == "gitlab") {
    link = `${repo.replace("gitlab", "gitlab.com")}/-/commit/${shaShort}`;
  } else {
    link = `${repo.replace("bitbucket", "bitbucket.org")}/commits/${shaShort}`;
  }

  return (
    <Link
      href={`https://${link}`}
      sx={{ ...sx, fontSize: "inherit", color: "inherit" }}
      {...rest}
    >
      #{sha.substring(0, 6)}
    </Link>
  );
}
