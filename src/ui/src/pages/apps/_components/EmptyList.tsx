import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import Typography from "@mui/material/Typography";
import emptyListSvg from "~/assets/images/empty-list.svg";

interface Props {
  primaryDesc?: string;
  primaryActionText?: string;
  primaryLink?: string;
  secondaryDesc?: string;
  secondaryLink?: string;
  secondaryActionText?: string;
  onBareAppClick?: () => void;
}

export default function EmptyList({
  primaryLink,
  primaryActionText,
  primaryDesc,
  secondaryLink,
  secondaryActionText,
  secondaryDesc,
  onBareAppClick,
}: Props) {
  return (
    <Box sx={{ textAlign: "center" }}>
      <Box
        component="img"
        src={emptyListSvg}
        alt="Empty app list"
        sx={{ margin: "0 auto" }}
      />
      <Typography sx={{ my: 6, fontSize: 18, color: "text.secondary" }}>
        Choose one of these options to get started
      </Typography>
      <Box
        sx={{
          display: "flex",
          flexDirection: { xs: "column", md: "row" },
          justifyContent: "center",
          alignItems: "center",
        }}
      >
        <Card
          sx={{
            backgroundColor: "container.paper",
            mr: { xs: 0, md: 2 },
            mb: { xs: 2, md: 0 },
            ml: 0,
            width: { xs: "100%", md: "auto" },
            maxWidth: { xs: "none", md: 340 },
            borderRadius: 1,
            flex: 1,
          }}
        >
          <CardHeader title="Private repositories" sx={{ pb: 2 }} />
          <Typography sx={{ mb: 4, color: "text.secondary" }}>
            {primaryDesc}
          </Typography>
          <CardFooter sx={{ textAlign: "center" }}>
            <Button variant="contained" color="secondary" href={primaryLink}>
              {primaryActionText}
            </Button>
          </CardFooter>
        </Card>
        <Typography component="span">or</Typography>
        <Card
          sx={{
            backgroundColor: "container.paper",
            ml: { xs: 0, md: 2 },
            mr: { md: onBareAppClick ? 2 : undefined },
            mt: { xs: 2, md: 0 },
            width: { xs: "100%", md: "auto" },
            maxWidth: { xs: "none", md: 340 },
            borderRadius: 1,
            flex: 1,
          }}
        >
          <CardHeader title="Public repositories" sx={{ pb: 2 }} />
          <Typography sx={{ mb: 4, color: "text.secondary" }}>
            {secondaryDesc}
          </Typography>
          <CardFooter sx={{ textAlign: "center" }}>
            <Button variant="contained" color="secondary" href={secondaryLink}>
              {secondaryActionText}
            </Button>
          </CardFooter>
        </Card>
        {onBareAppClick && (
          <>
            <Typography component="span">or</Typography>
            <Card
              sx={{
                backgroundColor: "container.paper",
                ml: { xs: 0, md: 2 },
                mt: { xs: 2, md: 0 },
                maxWidth: { xs: "none", md: 340 },
                width: { xs: "100%", md: "auto" },
                borderRadius: 1,
                flex: 1,
              }}
            >
              <CardHeader title="Bare app" sx={{ pb: 2 }} />
              <Typography sx={{ mb: 4, color: "text.secondary" }}>
                Deploy your websites by uploading zip files
              </Typography>
              <CardFooter sx={{ textAlign: "center" }}>
                <Button
                  variant="contained"
                  color="secondary"
                  onClick={e => {
                    e.preventDefault();
                    onBareAppClick?.();
                  }}
                >
                  Create bare app
                </Button>
              </CardFooter>
            </Card>
          </>
        )}
      </Box>
    </Box>
  );
}
