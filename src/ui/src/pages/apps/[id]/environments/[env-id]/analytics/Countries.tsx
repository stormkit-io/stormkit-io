import { useContext } from "react";
import WorldMap from "react-svg-worldmap";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { RootContext } from "~/pages/Root.context";
import { useFetchByCountries } from "./actions";
import { grey } from "@mui/material/colors";

interface Props {
  environment: Environment;
  domain?: Domain;
}

export default function TopReferrers({ environment, domain }: Props) {
  const { mode } = useContext(RootContext);
  const { countries, error, loading } = useFetchByCountries({
    envId: environment.id!,
    domainId: domain?.id,
  });

  const isDark = mode === "dark";

  return (
    <Card
      error={error}
      loading={loading}
      sx={{
        mt: 2,
        width: "100%",
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <CardHeader
        title="Countries"
        subtitle="The world map indicates the number of visitors per country."
      />
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          bgcolor: isDark ? "black" : grey[200],
          mb: 4,
        }}
      >
        <WorldMap
          backgroundColor={isDark ? "transparent" : grey[200]}
          borderColor={isDark ? "white" : "black"}
          color="white"
          size="responsive"
          valueSuffix="visits"
          richInteraction
          data={countries}
        />
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          Double click to zoom in and out.
        </Typography>
      </CardFooter>
    </Card>
  );
}
