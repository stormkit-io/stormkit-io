import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import { useFetchTopReferrers } from "./actions";
import { truncate } from "./helpers";

interface Props {
  environment: Environment;
  domainName?: string;
  domain: Domain;
}

export default function TopReferrers({ environment, domain }: Props) {
  const { referrers, error, loading } = useFetchTopReferrers({
    envId: environment.id!,
    domainId: domain?.id,
  });

  return (
    <Card
      error={error}
      loading={loading}
      sx={{
        width: { xs: "100%", md: "50%" },
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <CardHeader
        title="Referrers"
        subtitle="Top 50 referrers in the last 30 days."
      />
      <Box
        sx={{
          maxHeight: "300px",
          overflow: "auto",
          flex: 1,
        }}
      >
        {referrers.map(ref => (
          <CardRow
            key={ref.name}
            chipLabel={
              <Typography component="span">{ref.count.toString()} </Typography>
            }
          >
            <Typography component="span">{truncate(ref.name)}</Typography>
          </CardRow>
        ))}
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          Same domain referrers are excluded.
        </Typography>
      </CardFooter>
    </Card>
  );
}
