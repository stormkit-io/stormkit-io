import api from "~/utils/api/Api";

// revealEnvVars fetches the real (unmasked) environment variable values. The
// endpoint is restricted to team admins/owners and the reveal is audited.
export const revealEnvVars = ({
  envId,
}: {
  envId: string;
}): Promise<Record<string, string>> => {
  // An env with no variables serializes as JSON `null`; coalesce so callers
  // (and KeyValue's Object.keys) always get a usable map.
  return api
    .fetch<Record<string, string> | null>(`/v1/env/pull?envId=${envId}`)
    .then(vars => vars || {});
};

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

  // One directory per line; an empty textarea clears the list.
  let cacheDirs: string[] | undefined;

  if (values["build.cacheDirs"] !== undefined) {
    cacheDirs = values["build.cacheDirs"]
      .split("\n")
      .map(dir => dir.trim())
      .filter(Boolean);
  }

  const build: BuildConfig = {
    buildCmd: values["build.buildCmd"]?.trim() || "",
    serverCmd: values["build.serverCmd"]?.trim() || "",
    installCmd: values["build.installCmd"]?.trim() || "",
    distFolder: (values["build.distFolder"] || "").trim(),
    workDir: (values["build.workDir"] || "").trim(),
    headers: values["build.headers"]?.trim() || "",
    headersFile: values["build.headersFile"],
    redirectsFile: values["build.redirectsFile"],
    errorFile: values["build.errorFile"],
    apiFolder: values["build.apiFolder"],
    apiPathPrefix: values["build.apiPathPrefix"],
    previewLinks: values["build.previewLinks"] === "on",
    priorityPattern: values["build.priorityPattern"]?.trim() || "",
    statusChecks,
    redirects,
    cacheDirs,
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
    "build.priorityPattern": env.build.priorityPattern || "",
    "build.headers": env.build.headers || "",
    "build.headersFile": env.build.headersFile,
    "build.redirectsFile": env.build.redirectsFile,
    "build.apiFolder": env.build.apiFolder,
    "build.apiPathPrefix": env.build.apiPathPrefix,
    "build.buildCmd": env.build.buildCmd,
    "build.serverCmd": env.build.serverCmd,
    "build.installCmd": env.build.installCmd,
    "build.distFolder": env.build.distFolder,
    "build.workDir": env.build.workDir,
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
  "build.workDir"?: string;
}

// EnvUpdatePayload is a partial environment update: each config section sends
// only the keys it owns, and the API leaves any omitted field untouched. This
// keeps a section's save from overwriting fields it does not control — most
// importantly `envVars`, which only the environment-variables section sends.
export interface EnvUpdatePayload {
  name?: string;
  branch?: string;
  autoPublish?: boolean;
  autoDeploy?: boolean;
  autoDeployBranches?: string;
  autoDeployCommits?: string;
  previewLinks?: boolean;
  markdown?: boolean;
  priorityPattern?: string;
  buildCmd?: string;
  serverCmd?: string;
  installCmd?: string;
  distFolder?: string;
  workDir?: string;
  headers?: string;
  headersFile?: string;
  redirectsFile?: string;
  errorFile?: string;
  apiFolder?: string;
  apiPathPrefix?: string;
  statusChecks?: StatusCheck[];
  redirects?: Redirect[];
  envVars?: Record<string, string>;
  cacheDirs?: string[];
}

export interface FormValues {
  name?: string;
  branch?: string;
  autoDeploy?: AutoDeployValues;
  autoPublish?: "on" | "off";
  autoDeployBranches?: string;
  autoDeployCommits?: string;
  "build.priorityPattern"?: string;
  "build.statusChecks"?: string;
  "build.previewLinks"?: "on" | "off";
  "build.buildCmd"?: string;
  "build.serverCmd"?: string;
  "build.installCmd"?: string;
  "build.distFolder"?: string;
  "build.workDir"?: string;
  "build.cacheDirs"?: string;
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
  // payload holds only the keys to update; omitted fields are left unchanged.
  payload: EnvUpdatePayload;
  successMsg?: string;
  setLoading: (b: boolean) => void;
  setError: (s: string) => void;
  setSuccess: (s: string) => void;
  setRefreshToken: (t: number) => void;
}

export const updateEnvironment = ({
  envId,
  payload,
  setError,
  setLoading,
  setSuccess,
  setRefreshToken,
  successMsg = "The environment has been successfully updated.",
}: UpdateEnvironmentProps): Promise<void> => {
  if (("name" in payload && !payload.name) || ("branch" in payload && !payload.branch)) {
    setError("Environment and branch names are required.");
    return Promise.resolve();
  }

  setLoading(true);
  setError("");
  setSuccess("");

  // Send envId plus only the provided keys; drop undefined so a section can
  // include a field conditionally without clearing it server-side.
  const body: Record<string, unknown> = { envId };

  (Object.keys(payload) as (keyof EnvUpdatePayload)[]).forEach(key => {
    if (payload[key] !== undefined) {
      body[key] = payload[key];
    }
  });

  return api
    .put<{ status: boolean }>(`/v1/env`, body)
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

