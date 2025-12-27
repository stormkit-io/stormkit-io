import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import BoltIcon from "@mui/icons-material/Bolt";
import Dot from "~/components/Dot";
import IconBg from "~/components/IconBg";
import { getLogoForProvider, parseRepo } from "~/utils/helpers/providers";

interface Props {
  app: App;
  imageSize?: number;
}

export default function AppName({ app, imageSize = 20 }: Props) {
  if (app.isBare) {
    return (
      <Typography
        component="div"
        sx={{ display: "flex", alignItems: "center" }}
      >
        <IconBg
          sx={{
            width: imageSize,
            height: imageSize,
            mr: 1,
            bgcolor: "text.primary",
          }}
        >
          <BoltIcon sx={{ ml: 0, fontSize: 16, color: "container.paper" }} />
        </IconBg>
        {app.displayName}
      </Typography>
    );
  }

  const { repo, provider } = parseRepo(app.repo);
  const providerLogo = getLogoForProvider(provider);

  return (
    <Box
      sx={{
        display: "flex",
        alignItems: "center",
      }}
    >
      <Box
        component="img"
        sx={{
          display: "inline-block",
          mr: 1,
          width: imageSize,
        }}
        src={providerLogo}
        alt={provider}
      />

      <Typography component="div">
        {app.displayName}
        <Dot />
        <Typography component="span" color="text.secondary">
          {repo}
        </Typography>
      </Typography>
    </Box>
  );
}
