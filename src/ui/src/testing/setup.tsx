import { afterEach, vi } from "vitest";
import fetch, { Headers, Request, Response } from "node-fetch";
import nock from "nock";

(global as any).fetch = fetch;
(global as any).Headers = Headers;
(global as any).Request = Request;
(global as any).Response = Response;

declare const global: {
  NavigateMock: any;
};

global.NavigateMock = vi.fn();

vi.mock("react-router", async importOriginal => {
  const OriginalModule = await importOriginal<typeof import("react-router")>();

  return {
    ...OriginalModule,
    useNavigate: () => global.NavigateMock,
  };
});

vi.mock("react-router-dom", async importOriginal => {
  const OriginalModule = await importOriginal<
    typeof import("react-router-dom")
  >();

  return {
    ...OriginalModule,
    useNavigate: () => global.NavigateMock,
  };
});

vi.mock("@uiw/react-codemirror", () => ({
  default: ({ value }: { value: string }) => (
    <span data-testid="editor">{value}</span>
  ),
}));

vi.mock(import("@codemirror/lang-json"), async importOriginal => ({
  ...(await importOriginal()),
}));

interface RechartsProps {
  children: React.ReactElement;
  data: any;
}

window.ResizeObserver =
  window.ResizeObserver ||
  vi.fn().mockImplementation(() => ({
    disconnect: vi.fn(),
    observe: vi.fn(),
    unobserve: vi.fn(),
  }));

vi.mock("recharts", async importOriginal => {
  const OriginalModule = await importOriginal<typeof import("recharts")>();
  const fn =
    (testid: string) =>
    ({ children, data }: RechartsProps) => {
      return (
        <>
          <div data-testid={testid} style={{ width: "800px", height: "800px" }}>
            {JSON.stringify(data || "")}
          </div>
          <div>{children}</div>
        </>
      );
    };

  return {
    ...OriginalModule,
    AreaChart: fn("area-chart"),
    YAxis: fn("y-axis"),
    Area: fn("area"),
    Tooltip: fn("tooltip"),
    CartesianGrid: fn("cartesian-grid"),
    ResponsiveContainer: fn("responsive-container"),
  };
});

class MockDate extends Date {
  /**
   * Mock the function and return always the ISO 8601 format for tests.
   */
  toLocaleDateString() {
    return (
      this.getFullYear() +
      "-" +
      ("0" + (this.getMonth() + 1)).slice(-2) +
      "-" +
      ("0" + this.getDate()).slice(-2)
    );
  }
}

const div = document.createElement("div");
div.id = "side-bar-root";
document.body.appendChild(div);

// @ts-ignore
global.Date = MockDate;

afterEach(() => {
  nock.cleanAll();
  nock.disableNetConnect();
  nock.enableNetConnect("127.0.0.1");
});

// FAIL LOUDLY on unhandled promise rejections / errors
// @ts-ignore
process.on("unhandledRejection", reason => {
  // eslint-disable-next-line no-console
  console.log(`FAILED TO HANDLE PROMISE REJECTION`);
  throw reason;
});
