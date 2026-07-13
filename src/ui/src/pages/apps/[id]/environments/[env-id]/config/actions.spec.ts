import { describe, expect, it, vi } from "vitest";
import { buildFormValues, validateRedirects } from "./actions";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";

describe("~/pages/apps/[id]/environments/[env-id]/config/actions.tsx", () => {
  const app = mockApp();
  const env = mockEnvironments({ app })[0];

  describe("buildFormValues", () => {
    it("should match the default state", () => {
      const values = buildFormValues(env, document.createElement("form"));

      expect(values).toEqual({
        name: env.name,
        branch: env.branch,
        autoDeploy: "disabled",
        autoDeployBranches: undefined,
        autoPublish: "on",
        "build.redirects": undefined,
        "build.serverCmd": undefined,
        "build.statusChecks": undefined,
        "build.previewLinks": "on",
        "build.headersFile": env.build.headersFile,
        "build.redirectsFile": env.build.redirectsFile,
        "build.apiFolder": env.build.apiFolder,
        "build.buildCmd": env.build.buildCmd,
        "build.distFolder": env.build.distFolder,
        "build.vars": "BABEL_ENV=production\nNODE_ENV=production",
      });
    });

    it("should match the updated form values", () => {
      const form = document.createElement("form");

      const name = document.createElement("input");
      name.type = "text";
      name.name = "name";
      name.value = "my-new-env";

      const autoPublish = document.createElement("input");
      autoPublish.type = "text";
      autoPublish.name = "autoPublish";
      autoPublish.value = "off";

      const autoDeploy = document.createElement("input");
      autoDeploy.type = "text";
      autoDeploy.name = "autoDeploy";
      autoDeploy.value = "custom";

      const autoDeployBranches = document.createElement("input");
      autoDeployBranches.type = "text";
      autoDeployBranches.name = "autoDeployBranches";
      autoDeployBranches.value = "^(?!dependabot).+";

      form.appendChild(name);
      form.appendChild(autoPublish);
      form.appendChild(autoDeploy);
      form.appendChild(autoDeployBranches);

      const values = buildFormValues(env, form);

      expect(values).toEqual({
        name: "my-new-env",
        branch: env.branch,
        autoDeploy: "custom",
        autoDeployBranches: "^(?!dependabot).+",
        autoDeployCommits: "",
        autoPublish: "off",
        "build.redirects": undefined,
        "build.serverCmd": undefined,
        "build.statusChecks": undefined,
        "build.previewLinks": "on",
        "build.headersFile": env.build.headersFile,
        "build.redirectsFile": env.build.redirectsFile,
        "build.apiFolder": env.build.apiFolder,
        "build.buildCmd": env.build.buildCmd,
        "build.distFolder": env.build.distFolder,
        "build.vars": "BABEL_ENV=production\nNODE_ENV=production",
      });
    });
  });

  describe("validateRedirects", () => {
    const setError = vi.fn();
    let redirects = `{}`;

    expect(validateRedirects(redirects, setError)).toBe(false);
    expect(setError).toHaveBeenCalledWith(
      "Invalid format for redirects: expected an array of objects."
    );

    setError.mockClear();
    redirects = `[ 
      { "from": 1 } 
    ]`;

    expect(validateRedirects(redirects, setError)).toBe(false);
    expect(setError).toHaveBeenCalledWith(
      "Invalid format for redirects: `from` needs to be type of string."
    );

    setError.mockClear();
    redirects = `[ 
      { "from": "/", "to": 2 } 
    ]`;

    expect(validateRedirects(redirects, setError)).toBe(false);
    expect(setError).toHaveBeenCalledWith(
      "Invalid format for redirects: `to` needs to be type of string."
    );

    setError.mockClear();
    redirects = `[ 
      { "from": "/", "to": "/", "status": 400 } 
    ]`;

    expect(validateRedirects(redirects, setError)).toBe(false);
    expect(setError).toHaveBeenCalledWith(
      "Invalid format for redirects: `status` needs to be either 200 or 3xx."
    );

    setError.mockClear();
    redirects = `[ 
      { "from": "/", "to": "/", "status": 200, "assets": "true" } 
    ]`;

    expect(validateRedirects(redirects, setError)).toBe(false);
    expect(setError).toHaveBeenCalledWith(
      "Invalid format for redirects: `assets` needs to be either true, false or undefined."
    );

    setError.mockClear();
    redirects = `[ 
      { "from": "/", "to": "/", "status": 200, "assets": true, "hosts": "abc.com" } 
    ]`;

    expect(validateRedirects(redirects, setError)).toBe(false);
    expect(setError).toHaveBeenCalledWith(
      "Invalid format for redirects: `hosts` needs an array of strings."
    );

    setError.mockClear();
    redirects = `[ 
      { "from": "/", "to": "/", "status": 200, "assets": true, "hosts": ["abc.com"] } 
    ]`;

    expect(validateRedirects(redirects, setError)).toBe(true);
    expect(setError).not.toHaveBeenCalled();
  });
});
