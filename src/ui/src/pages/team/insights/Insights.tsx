import { useContext } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TrendingUpIcon from "@mui/icons-material/TrendingUp";
import TrendingDownIcon from "@mui/icons-material/TrendingDown";
import TrendingFlatIcon from "@mui/icons-material/TrendingFlat";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import EmptyPage from "~/components/EmptyPage";
import { formatNumber } from "~/utils/helpers/string";
import { calculateChange } from "~/utils/helpers/stats";
import { useFetchTopDomains } from "./actions";

export default function Insights() {
  const { teams } = useContext(AuthContext);
  const selectedTeam = useSelectedTeam({ teams });
  const { domains, loading, error, paymentRequired } = useFetchTopDomains({
    teamId: selectedTeam?.id,
  });

  return (
    <Box maxWidth="lg" sx={{ width: "100%" }}>
      <Card
        sx={{ width: "100%" }}
        loading={loading}
        error={error}
        info={
          !loading &&
          !error &&
          !domains?.length &&
          !paymentRequired &&
          "No domains founds. Data is fetched daily."
        }
      >
        <CardHeader
          title="Requests"
          subtitle={
            <>
              Top domains by requests over the last 30 days period.
              <br />
              Comparisons are made against previous 30 days.
            </>
          }
          actions={
            !paymentRequired &&
            !loading && (
              <Typography>
                {new Date(
                  Date.now() - 30 * 24 * 60 * 60 * 1000
                ).toLocaleDateString("en", {
                  year: "numeric",
                  month: "long",
                  day: "2-digit",
                })}
                {" - "}
                {new Date().toLocaleDateString("en", {
                  year: "numeric",
                  month: "long",
                  day: "2-digit",
                })}
              </Typography>
            )
          }
        />
        {paymentRequired && <EmptyPage paymentRequired />}
        <Box sx={{ maxHeight: "450px", overflow: "auto" }}>
          {domains?.map((d, i) => {
            const change = calculateChange(d.current, d.previous);
            const Icon = !change
              ? TrendingFlatIcon
              : change > 0
              ? TrendingUpIcon
              : TrendingDownIcon;

            return (
              <CardRow
                key={d.id}
                actions={
                  <Box
                    sx={{
                      display: "flex",
                      alignItems: "center",
                    }}
                  >
                    <Icon
                      sx={{ mr: 1 }}
                      fontSize="small"
                      color={
                        !change ? "primary" : change > 0 ? "success" : "error"
                      }
                    />
                    <Box sx={{ display: "flex" }}>
                      <Typography>{formatNumber(d.current)}</Typography>
                      <Typography
                        color="text.secondary"
                        sx={{ display: " inline-block", ml: 1 }}
                      >
                        {!change
                          ? ""
                          : change > 0
                          ? `+${formatNumber(d.current - d.previous)}`
                          : `-${formatNumber(d.previous - d.current)}`}
                      </Typography>
                    </Box>
                  </Box>
                }
              >
                <Typography>
                  <Typography
                    component="span"
                    color="text.secondary"
                    sx={{ display: "inline-block", width: "25px" }}
                  >
                    {i + 1}.
                  </Typography>{" "}
                  {d.domainName}
                </Typography>
              </CardRow>
            );
          })}
        </Box>
        <CardFooter />
      </Card>
    </Box>
  );
}
