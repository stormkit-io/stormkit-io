import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import CardRow from "~/components/CardRow";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import UpgradeButton from "~/components/UpgradeButton";
import { capitalize } from "~/utils/helpers/string";
import api from "~/utils/api/Api";
import { RootContext } from "../Root.context";

export default function Subscription() {
  const { details, setRefreshToken } = useContext(RootContext);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [loading, setLoading] = useState<"submit" | "cancel">();
  const license = details?.license;
  const isEnterprise = license?.edition === "enterprise";

  return (
    <>
      <Card
        sx={{ backgroundColor: "container.transparent", mb: 4 }}
        contentPadding={false}
      >
        <CardHeader
          title="Subscription details"
          subtitle="Information about your Stormkit subscription"
        />
        <CardRow>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            <Typography variant="h3" color="text.secondary" sx={{ width: 120 }}>
              Plan
            </Typography>
            <Typography>
              {capitalize(license?.edition) || "Community"}
            </Typography>
          </Box>
        </CardRow>
        <CardRow>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            <Typography variant="h3" color="text.secondary" sx={{ width: 120 }}>
              Purchased Seats
            </Typography>
            <Typography>
              {license?.seats === -1 ? "Unlimited" : license?.seats}
            </Typography>
          </Box>
        </CardRow>
        <CardRow>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            <Typography variant="h3" color="text.secondary" sx={{ width: 120 }}>
              Remaining seats
            </Typography>
            <Typography>
              {license?.seats === -1 ? "Unlimited" : license?.remaining}
            </Typography>
          </Box>
        </CardRow>
        <CardFooter>
          <UpgradeButton text="Manage subscription" fullWidth={false} />
        </CardFooter>
      </Card>
      <Card
        sx={{ backgroundColor: "container.transparent" }}
        component="form"
        success={success}
        error={error}
        onSubmit={e => {
          e.preventDefault();
          const form = e.target as HTMLFormElement;
          const data = Object.fromEntries(
            new FormData(form).entries(),
          ) as Record<string, string>;

          if (!data.licenseKey) {
            return setError("License key is a required field.");
          }

          setLoading("submit");

          api
            .post("/admin/license", { key: data.licenseKey })
            .then(() => {
              form?.reset();
              setRefreshToken?.(Date.now());
              setError("");
              setSuccess("License activated successfully.");
            })
            .catch(res => {
              setSuccess("");

              if (res.status === 400) {
                setError("The license is either invalid or expired.");
              } else {
                setError("An error occurred while activating the license.");
              }
            })
            .finally(() => {
              setLoading(undefined);
            });
        }}
      >
        <CardHeader
          title="License key"
          subtitle="Activate your Stormkit Enterprise license by pasting your license key below"
        />

        <Box sx={{ flex: 1, mb: 4 }}>
          <TextField
            label="License key"
            placeholder={
              "You can find your license key by connecting your account to app.stormkit.io and purchasing a license."
            }
            variant="filled"
            name="licenseKey"
            defaultValue={""}
            fullWidth
            multiline
            minRows={4}
            maxRows={4}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />
        </Box>

        <CardFooter>
          <Button
            type="button"
            variant="text"
            color="info"
            disabled={!isEnterprise}
            loading={loading === "cancel"}
            sx={{ mr: 2 }}
            onClick={() => {
              api
                .post("/admin/license")
                .then(() => {
                  setRefreshToken?.(Date.now());
                  setError("");
                  setSuccess("License removed successfully.");
                })
                .catch(res => {
                  setSuccess("");

                  if (res.status === 400) {
                    setError("The license is either invalid or expired.");
                  } else {
                    setError("An error occurred while activating the license.");
                  }
                })
                .finally(() => {
                  setLoading(undefined);
                });
            }}
          >
            Remove license
          </Button>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            loading={loading === "submit"}
          >
            {license?.edition === "enterprise" ? "Update license" : "Activate"}
          </Button>
        </CardFooter>
      </Card>
    </>
  );
}
