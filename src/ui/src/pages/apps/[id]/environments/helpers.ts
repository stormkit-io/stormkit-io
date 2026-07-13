import { toArray } from "~/utils/helpers/array";

export const prepareBuildObject = (
  values: Record<string, string>
): BuildConfig => {
  const vars: Record<string, string> = {};

  const keys = toArray<string>(values["build.vars.keys"]);
  const vals = toArray<string>(values["build.vars.values"]);

  keys.forEach((key, i) => {
    if (key.trim() !== "") {
      vars[key.trim()] = vals[i].trim().replace(/^['"]+|['"]+$/g, "");
    }
  });

  const build: BuildConfig = {
    buildCmd: values["build.buildCmd"],
    distFolder: values["build.distFolder"] || "",
    vars,
  };

  return build;
};
