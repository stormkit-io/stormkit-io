import { FormEventHandler, useState } from "react";
import api from "~/utils/api/Api";

export const computeAutoDeployValue = (env?: Environment): AutoDeployValues => {
  if (!env) {
    return "all";
  }

  if (env.autoDeployBranches) {
    return "custom";
  }

  if (env.autoDeployCommits) {
    return "custom_commit";
  }

  if (env.autoDeploy) {
    return "all";
  }

  return "disabled";
};

export const prepareBuildObject = (values: FormValues): BuildConfig => {
  const vars: Record<string, string> = {};

  values["build.vars"]?.split("\n").forEach(line => {
    if (line.indexOf("=") > 0) {
      const indexOfEqual = line.indexOf("=");

      vars[line.slice(0, indexOfEqual).trim()] = line
        .slice(indexOfEqual + 1)
        .trim();
    }
  });

  let redirects: Redirect[] | undefined;

  if (values["build.redirects"]) {
    try {
      redirects = JSON.parse(values["build.redirects"]);
    } catch {}
  }

  let statusChecks: StatusCheck[] | undefined;

  if (values["build.statusChecks"]) {
    try {
      statusChecks = JSON.parse(values["build.statusChecks"]);
    } catch {}
  }

  const build: BuildConfig = {
    buildCmd: values["build.buildCmd"]?.trim() || "",
    serverCmd: values["build.serverCmd"]?.trim() || "",
    installCmd: values["build.installCmd"]?.trim() || "",
    distFolder: (values["build.distFolder"] || "").trim(),
    headers: values["build.headers"]?.trim() || "",
    headersFile: values["build.headersFile"],
    redirectsFile: values["build.redirectsFile"],
    errorFile: values["build.errorFile"],
    apiFolder: values["build.apiFolder"],
    apiPathPrefix: values["build.apiPathPrefix"],
    previewLinks: values["build.previewLinks"] === "on",
    statusChecks,
    redirects,
    vars,
  };

  return build;
};

export const buildFormValues = (
  env: Environment,
  form: HTMLFormElement,
  controlled?: ControlledFormValues,
): FormValues => {
  let values = Object.fromEntries(new FormData(form).entries());

  // This is for controlled values, such as Switches.
  if (controlled) {
    values = { ...values, ...controlled };
  }

  if (typeof values.autoPublish === "undefined") {
    values.autoPublish = env.autoPublish ? "on" : "off";
  }

  if (typeof values["build.previewLinks"] === "undefined") {
    values["build.previewLinks"] =
      env.build.previewLinks !== false ? "on" : "off";
  }

  // Normalize autoDeploy values
  if (values.autoDeploy) {
    if (values.autoDeploy !== "custom") {
      values.autoDeployBranches = "";
    }

    if (values.autoDeploy !== "custom_commit") {
      values.autoDeployCommits = "";
    }
  }

  return {
    name: env.name,
    branch: env.branch,
    autoPublish: env.autoPublish ? "on" : "off",
    autoDeploy: computeAutoDeployValue(env),
    autoDeployBranches: env.autoDeployBranches || undefined,
    autoDeployCommits: env.autoDeployCommits || undefined,
    "build.statusChecks": JSON.stringify(env.build.statusChecks),
    "build.previewLinks": env.build.previewLinks ? "on" : "off",
    "build.headersFile": env.build.headersFile,
    "build.redirectsFile": env.build.redirectsFile,
    "build.apiFolder": env.build.apiFolder,
    "build.apiPathPrefix": env.build.apiPathPrefix,
    "build.buildCmd": env.build.buildCmd,
    "build.serverCmd": env.build.serverCmd,
    "build.installCmd": env.build.installCmd,
    "build.distFolder": env.build.distFolder || env.build.serverFolder,
    "build.redirects": JSON.stringify(env.build.redirects),
    "build.vars": Object.keys(env.build?.vars || {})
      .filter(key => env.build.vars[key])
      .map(key => `${key}=${env.build.vars[key]}`)
      .join("\n"),
    ...values,
  };
};

export type AutoDeployValues = "disabled" | "all" | "custom" | "custom_commit";

interface ControlledFormValues {
  autoPublish?: "on" | "off";
  "build.previewLinks"?: "on" | "off";
  "build.redirects"?: string;
  "build.headers"?: string;
  "build.statusChecks"?: string;
}

export interface FormValues {
  name?: string;
  branch?: string;
  autoDeploy?: AutoDeployValues;
  autoPublish?: "on" | "off";
  autoDeployBranches?: string;
  autoDeployCommits?: string;
  "build.statusChecks"?: string;
  "build.previewLinks"?: "on" | "off";
  "build.buildCmd"?: string;
  "build.serverCmd"?: string;
  "build.installCmd"?: string;
  "build.distFolder"?: string;
  "build.headers"?: string;
  "build.headersFile"?: string;
  "build.errorFile"?: string;
  "build.redirects"?: string;
  "build.redirectsFile"?: string;
  "build.apiFolder"?: string;
  "build.apiPathPrefix"?: string;
  "build.vars"?: string; // This is the textarea version
  "build.vars[key]"?: string; // This is the key value version
  "build.vars[value]"?: string; // This is the key value version
}

interface UpdateEnvironmentProps {
  app: App;
  envId: string;
  values: FormValues;
  successMsg?: string;
  setLoading: (b: boolean) => void;
  setError: (s: string) => void;
  setSuccess: (s: string) => void;
  setRefreshToken: (t: number) => void;
}

export const updateEnvironment = ({
  app,
  envId,
  values,
  setError,
  setLoading,
  setSuccess,
  setRefreshToken,
  successMsg = "The environment has been successfully updated.",
}: UpdateEnvironmentProps): Promise<void> => {
  const build = prepareBuildObject(values);

  if (!values.name || !values.branch) {
    setError("Environment and branch names are required.");
    return Promise.resolve();
  }

  setLoading(true);
  setError("");
  setSuccess("");

  return api
    .put<{ status: boolean }>(`/app/env`, {
      id: envId,
      build,
      appId: app.id,
      env: values.name,
      branch: values.branch,
      autoPublish: values.autoPublish === "on",
      autoDeploy: values.autoDeploy !== "disabled",
      autoDeployBranches: values.autoDeployBranches,
      autoDeployCommits: values.autoDeployCommits,
    })
    .then(() => {
      setSuccess(successMsg);
      setRefreshToken(Date.now());
    })
    .catch(async res => {
      if (typeof res === "string") {
        return setError(res);
      }

      try {
        const jsonData = await res.json();
        setError(jsonData.error);
      } catch (e) {
        setError(`"Error: ${(await res?.body()) || res}`);
      }
    })
    .finally(() => {
      setLoading(false);
    });
};

interface InsertEnvironmentProps {
  app: App;
  values: FormValues;
}

interface InsertEnvironmentReturnValue {
  envId: string;
}

export const insertEnvironment = ({
  app,
  values,
}: InsertEnvironmentProps): Promise<InsertEnvironmentReturnValue> => {
  const { name, branch, autoDeployBranches, autoDeploy } = values;
  const build = prepareBuildObject(values);

  if (!name || !branch) {
    return new Promise((_, reject) => {
      reject("Environment and branch names are required.");
    });
  }

  return api.post<{ envId: string }>(`/app/env`, {
    appId: app.id,
    env: name,
    branch,
    build,
    autoPublish: values.autoPublish === "on",
    autoDeploy: autoDeploy !== "disabled",
    autoDeployBranches: autoDeployBranches || null,
  });
};

interface DeleteEnvironmentProps {
  app: App;
  environment: Environment;
}

export const deleteEnvironment = ({
  app,
  environment,
}: DeleteEnvironmentProps): Promise<void> => {
  const name = environment?.env;

  if (!name) {
    return Promise.reject();
  }

  return api.delete(`/app/env`, {
    appId: app.id,
    env: name,
  });
};

export const validateRedirects = (
  redirects: string,
  setError: (s: string) => void,
) => {
  if (!redirects) {
    return true;
  }

  try {
    const parsed = JSON.parse(redirects) as Redirect[];

    if (!Array.isArray(parsed)) {
      setError("Invalid format for redirects: expected an array of objects.");
      return false;
    }

    const availableStatuses = [200, 300, 301, 302, 303, 304, 305, 306, 307];

    for (const redirect of parsed) {
      if (typeof redirect.from !== "string") {
        setError(
          "Invalid format for redirects: `from` needs to be type of string.",
        );

        return false;
      }

      if (typeof redirect.to !== "string") {
        setError(
          "Invalid format for redirects: `to` needs to be type of string.",
        );

        return false;
      }

      if (redirect.status && !availableStatuses.includes(redirect.status)) {
        setError(
          "Invalid format for redirects: `status` needs to be either 200 or 3xx.",
        );

        return false;
      }

      if (redirect.assets && typeof redirect.assets !== "boolean") {
        setError(
          "Invalid format for redirects: `assets` needs to be either true, false or undefined.",
        );

        return false;
      }

      if (redirect.hosts) {
        if (!Array.isArray(redirect.hosts)) {
          setError(
            "Invalid format for redirects: `hosts` needs an array of strings.",
          );

          return false;
        }

        for (const host of redirect.hosts) {
          if (typeof host !== "string") {
            setError(
              "Invalid format for redirects: `hosts` needs an array of strings.",
            );

            return false;
          }
        }
      }
    }
  } catch {
    return false;
  }

  return true;
};

interface SubmitHandlerProps {
  app: App;
  env: Environment;
  setRefreshToken: (v: number) => void;
  controlled?: ControlledFormValues;
}

export const useSubmitHandler = ({
  env,
  app,
  setRefreshToken,
  controlled,
}: SubmitHandlerProps) => {
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [isLoading, setLoading] = useState(false);

  const handler: FormEventHandler = e => {
    e.preventDefault();

    const values: FormValues = buildFormValues(
      env,
      e.target as HTMLFormElement,
      controlled,
    );

    if (!validateRedirects(values["build.redirects"] || "", setError)) {
      setError("Invalid redirects format.");
      return Promise.resolve();
    }

    return updateEnvironment({
      app,
      envId: env.id!,
      values,
      setError,
      setLoading,
      setSuccess,
      setRefreshToken,
    });
  };

  return { submitHandler: handler, error, isLoading, success };
};
