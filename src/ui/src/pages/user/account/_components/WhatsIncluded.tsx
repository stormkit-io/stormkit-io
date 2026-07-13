import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import CheckIcon from "@mui/icons-material/Check";

const limits: Record<SubscriptionName, { fns: string; bandwidth: string }> = {
  free: {
    fns: "500k",
    bandwidth: "50GB",
  },
  starter: {
    fns: "2.5m",
    bandwidth: "500GB",
  },
  medium: {
    fns: "5m",
    bandwidth: "1TB",
  },
  enterprise: {
    fns: "10m",
    bandwidth: "2TB",
  },
  "self-hosted": {
    fns: "Unlimited",
    bandwidth: "Unlimited",
  },
};

interface Feature {
  text: string;
  included: boolean;
}

export const includedFeatures = (tier: SubscriptionName, edition?: Edition) => {
  const shared: Feature[] = [
    { included: true, text: "Unlimited seats" },
    { included: true, text: "Unlimited teams" },
    {
      included: true,
      text: `${limits[tier]?.fns || limits.starter.fns} serverless invocations`,
    },
    { included: true, text: "Unlimited projects" },
    {
      included: true,
      text: `${limits[tier]?.bandwidth || limits.starter.bandwidth} bandwidth`,
    },
    { included: true, text: "30 days deployment retention" },
  ];

  if (tier.startsWith("self-hosted")) {
    return [
      {
        included: true,
        text: "Unlimited usage",
      },
      {
        included: true,
        text: "Monitoring",
      },
      {
        included: edition === "premium",
        text: "Audit Logs",
      },
      {
        included: edition === "premium",
        text: "Custom TLS Certificates",
      },
      // {
      //   included: edition === "premium",
      //   text: "IP Limiting",
      // },
      {
        included: edition === "premium",
        text: "Premium support",
      },
      {
        included: edition === "premium",
        text: "Custom features",
      },
    ];
  }

  return [...shared, { included: true, text: "All features" }];
};

interface Props {
  tier: SubscriptionName;
  edition?: Edition;
}

export default function WhatsIncluded({ tier, edition }: Props) {
  return (
    <Box sx={{ display: "flex", mt: 2 }}>
      <Grid
        container
        sx={{ width: "100%", textAlign: "left" }}
        rowSpacing={{ xs: 2 }}
      >
        {includedFeatures(tier, edition).map((feature, index) => (
          <Grid
            key={index}
            item
            xs={12}
            md={6}
            sx={{ display: "flex", alignItems: "center" }}
          >
            <CheckIcon sx={{ fill: "green", mr: 2, ml: 0 }} /> {feature.text}
          </Grid>
        ))}
      </Grid>
    </Box>
  );
}
