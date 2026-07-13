import { render, RenderResult } from "@testing-library/react";
import Sha from "./Sha";

interface Props {
  repo: string;
  provider: Provider;
  sha: string;
}

describe("~/shared/deployments/Sha.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ repo, provider, sha }: Props) => {
    wrapper = render(<Sha repo={repo} provider={provider} sha={sha} />);
  };

  test.each`
    repo                                       | provider       | sha
    ${"github/stormkit-io/app-stormkit-io"}    | ${"github"}    | ${"c8b80debdde397405b787cc4ddffcb22867586b4"}
    ${"gitlab/stormkit-io/app-stormkit-io"}    | ${"gitlab"}    | ${"c8b80debdde397405b787cc4ddffcb22867586b4"}
    ${"bitbucket/stormkit-io/app-stormkit-io"} | ${"bitbucket"} | ${"c8b80debdde397405b787cc4ddffcb22867586b4"}
  `(
    "should display the correct URL for the $provider",
    ({ provider, repo, sha }) => {
      createWrapper({ repo, provider, sha });
      const short = sha.substring(0, 6);
      const element = wrapper.getByText(`#${short}`);
      expect(element).toBeTruthy();
      expect(element.getAttribute("href")).toContain(provider);
      expect(element.getAttribute("href")).toContain(short);
    }
  );

  test("should return empty string when sha is missing", () => {
    createWrapper({ repo: "", provider: "github", sha: "" });
    expect(wrapper.container.innerHTML).toBe("");
  });
});
