import type { RenderResult } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import AuditMessage from "./AuditMessage";

interface Props {
  audit: Audit;
}

describe("~/shared/feed/AuditMessage.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ audit }: Props) => {
    wrapper = render(<AuditMessage audit={audit} />);
  };

  const diffs: Record<string, Diff> = {
    "CREATE:DOMAIN": {
      old: {},
      new: { domainName: "app.stormkit.io" },
    },
    "UPDATE:DOMAIN": {
      old: { domainName: "app.stormkit.io" },
      new: {
        domainName: "app.stormkit.io",
        domainCertKey: "my-key",
        domainCertValue: "my-cert",
      },
    },
    "DELETE:DOMAIN": {
      old: { domainName: "app.stormkit.io" },
      new: {},
    },
    "CREATE:ENV": {
      old: {},
      new: { envName: "staging" },
    },
    "UPDATE:ENV": {
      old: {},
      new: { envName: "staging" },
    },
    "DELETE:ENV": {
      old: { envName: "staging" },
      new: {},
    },
    "CREATE:APP": {
      old: {},
      new: { appName: "sample-project" },
    },
    "UPDATE:APP": {
      old: {},
      new: { appName: "sample-project" },
    },
    "DELETE:APP": {
      old: { appName: "sample-project" },
      new: {},
    },
    "CREATE:SNIPPET": {
      old: {},
      new: { snippets: ["a", "b"] },
    },
    "UPDATE:SNIPPET": {
      old: {},
      new: {},
    },
    "DELETE:SNIPPET": {
      old: { snippets: ["2", "5"] },
      new: {},
    },
    "UPDATE:AUTHWALL": {
      old: {
        authWallStatus: "off",
      },
      new: {
        authWallStatus: "dev",
      },
    },
    "CREATE:AUTHWALL": {
      old: {},
      new: {
        authWallCreateLoginEmail: "joe@doe.org",
        authWallCreateLoginID: "2",
      },
    },
    "DELETE:AUTHWALL": {
      old: {
        authWallDeleteLoginIDs: "2,3",
      },
      new: {},
    },
    "UPDATE:DEPLOYMENT": {
      old: {},
      new: { deploymentId: "8241" },
    },
  };

  it.each`
    action                 | expected                                                          | diff
    ${"CREATE:DOMAIN"}     | ${"Added app.stormkit.io domain to production environment"}       | ${null}
    ${"UPDATE:DOMAIN"}     | ${"Added custom certificate to app.stormkit.io"}                  | ${null}
    ${"DELETE:DOMAIN"}     | ${"Removed app.stormkit.io domain from production environment"}   | ${null}
    ${"CREATE:ENV"}        | ${"Created the staging environment"}                              | ${null}
    ${"UPDATE:ENV"}        | ${"Updated the staging environment"}                              | ${null}
    ${"DELETE:ENV"}        | ${"Removed the staging environment"}                              | ${null}
    ${"CREATE:APP"}        | ${"Created the sample-project application"}                       | ${null}
    ${"UPDATE:APP"}        | ${"Updated the sample-project application"}                       | ${null}
    ${"DELETE:APP"}        | ${"Deleted the sample-project application"}                       | ${null}
    ${"CREATE:SNIPPET"}    | ${"Created 2 new snippets in production environment"}             | ${null}
    ${"UPDATE:SNIPPET"}    | ${"Updated 1 snippet in production environment"}                  | ${null}
    ${"DELETE:SNIPPET"}    | ${"Deleted 2 snippets in production environment"}                 | ${null}
    ${"UPDATE:AUTHWALL"}   | ${"Enabled auth wall in production environment"}                  | ${null}
    ${"CREATE:AUTHWALL"}   | ${"Created new auth login for production environment"}            | ${null}
    ${"DELETE:AUTHWALL"}   | ${"Deleted auth login from production environment"}               | ${null}
    ${"CREATE:SCHEMA"}     | ${"Created schema for production environment"}                    | ${null}
    ${"DELETE:SCHEMA"}     | ${"Deleted schema from production environment"}                   | ${null}
    ${"UPDATE:DEPLOYMENT"} | ${"Manually published deployment 8241 to production environment"} | ${null}
    ${"UPDATE:DEPLOYMENT"} | ${"Auto-published deployment 8241 to production environment"}     | ${{ old: {}, new: { deploymentId: "8241", autoPublished: true } }}
    ${"UPDATE:DEPLOYMENT"} | ${"Restarted deployment 8241 in production environment"}          | ${{ old: {}, new: { deploymentId: "8241", restarted: true } }}
  `(
    "displays the correct message for $action: $expected",
    ({ action, expected, diff }) => {
      const audit: Audit = {
        id: "1",
        action,
        appId: "1",
        envId: "1",
        envName: "production",
        userDisplay: "jdoe",
        timestamp: 1723501214,
        diff: diff || diffs[action],
      };

      createWrapper({ audit });

      expect(wrapper.container.textContent).toContain(expected);
    },
  );
});
