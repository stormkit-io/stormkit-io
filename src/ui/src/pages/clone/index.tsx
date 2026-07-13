import { useEffect } from "react";
import InfoBox from "~/components/InfoBox";
import Spinner from "~/components/Spinner";
import { useClone } from "./actions";
import qs from "query-string";
import { useNavigate } from "react-router";

export default function Clone() {
  const { template } = qs.parse(location.search);
  const { loading, error } = useClone(template);
  const navigate = useNavigate();

  useEffect(() => {
    if (!loading && !error) {
      navigate("/");
    }
  }, [loading, error]);

  return (
    <>
      {!loading && error && <InfoBox type={InfoBox.ERROR}>{error}</InfoBox>}
      {loading && (
        <div className="flex w-full justify-center mt-4 content-center">
          <div className="flex flex-col justify-center">
            <div>
              <InfoBox type="default">
                We are creating repository in Github and Stormkit
              </InfoBox>
            </div>
            <div className="flex items-center justify-center mt-4">
              <Spinner primary />
            </div>
          </div>
        </div>
      )}
    </>
  );
}
