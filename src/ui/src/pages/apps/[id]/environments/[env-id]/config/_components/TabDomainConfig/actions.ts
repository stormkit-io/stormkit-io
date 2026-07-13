import { useEffect, useState } from "react";
import api from "~/utils/api/Api";
export { useFetchDomains } from "~/shared/domains/actions";

interface SetDomainProps {
  appId: string;
  envId: string;
  values: { domain: string };
}

export const setDomain = ({
  appId,
  envId,
  values,
}: SetDomainProps): Promise<void> => {
  return api.post("/v1/domains", {
    appId,
    envId,
    domain: values.domain.trim(),
  });
};

interface DeleteDomainProps {
  appId: string;
  envId: string;
  domainId: string;
}

export const deleteDomain = ({
  appId,
  envId,
  domainId,
}: DeleteDomainProps): Promise<void> => {
  return api.delete(
    `/v1/domains?appId=${appId}&domainId=${domainId}&envId=${envId}`
  );
};

interface DomainLookupProps {
  domainId: string;
  appId: string;
  envId: string;
  refreshToken: number;
}

export const useDomainLookup = ({
  domainId,
  appId,
  envId,
  refreshToken,
}: DomainLookupProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [info, setDomainsInfo] = useState<DomainLookup>();

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<DomainLookup>(
        `/domains/lookup?appId=${appId}&envId=${envId}&domainId=${domainId}`
      )
      .then(res => {
        if (res.domainName) {
          setDomainsInfo(res);
        }
      })
      .catch(() => {
        setError("Something went wrong file fetching domain information.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, domainId, refreshToken]);

  return { loading, error, info };
};

interface UpdateCustomCertProps {
  setLoading: (v: boolean) => void;
  setError: (e?: string) => void;
  setSuccess: (e?: string) => void;
  certKey?: string;
  certValue?: string;
  appId: string;
  envId: string;
  domainId: string;
}

export const updateCustomCert = ({
  setLoading,
  setError,
  setSuccess,
  certKey,
  certValue,
  appId,
  envId,
  domainId,
}: UpdateCustomCertProps) => {
  if (!certValue?.includes("-----BEGIN CERTIFICATE-----")) {
    setError("Certificate must be PEM encoded.");
    return Promise.resolve(false);
  }

  if (!certKey?.includes("-----BEGIN PRIVATE KEY-----")) {
    setError("Private key must be PEM encoded.");
    return Promise.resolve(false);
  }

  setLoading(true);
  setError(undefined);

  return api
    .put("/v1/domains/cert", { appId, envId, domainId, certKey, certValue })
    .then(() => {
      setSuccess(
        "Certificate was saved successfully. It will be automatically applied for new requests."
      );

      return true;
    })
    .catch(async res => {
      console.log(res);
      let error = "";

      try {
        const data = await res.json();
        error = data.error;
      } catch {}

      setError(
        res.status === 400
          ? error
          : "Something went wrong while saving the custom certificate for the the domain."
      );
    })
    .finally(() => {
      setLoading(false);
    });
};

interface UpdateCustomCertProps {
  setLoading: (v: boolean) => void;
  setError: (e?: string) => void;
  setSuccess: (e?: string) => void;
  certKey?: string;
  certValue?: string;
  appId: string;
  envId: string;
  domainId: string;
}

export const deleteCustomCert = ({
  setLoading,
  setError,
  setSuccess,
  appId,
  envId,
  domainId,
}: UpdateCustomCertProps) => {
  setLoading(true);
  setError(undefined);

  return api
    .delete(`/v1/domains/cert?appId=${appId}&envId=${envId}&domainId=${domainId}`)
    .then(() => {
      setSuccess(
        "Custom certificate was removed. A new certificate will be issued automatically."
      );

      return true;
    })
    .catch(async res => {
      let error = "";

      try {
        const data = await res.json();
        error = data.error;
      } catch {}

      setError(
        res.status === 400
          ? error
          : "Something went wrong while deleting the custom certificate for the the domain."
      );
    })
    .finally(() => {
      setLoading(false);
    });
};
