import type { RenderResult } from "@testing-library/react";
import type { Metrics, Sample } from "./actions";
import nock from "nock";
import { describe, it, expect, beforeEach } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import Health from "./Health";
import { formatBytes, formatUptime } from "./actions";

const defaultMetrics: Metrics = {
  retentionHours: 24,
  pool: {
    open: 3,
    inUse: 1,
    idle: 2,
    waitCount: 0,
    waitDurationMs: 0,
    maxOpenConnections: 10,
  },
  dependencies: {
    ts: 1754100000,
    postgres: {
      reachable: true,
      latencyMs: 1.25,
      databaseBytes: 5 * 1024 * 1024 * 1024,
      connections: 12,
      maxConnections: 100,
      largestTables: [{ name: "public.access_logs", bytes: 2 * 1024 ** 3 }],
    },
    redis: {
      reachable: true,
      latencyMs: 0.5,
      usedMemoryBytes: 12 * 1024 * 1024,
    },
  },
  machines: [
    {
      host: "node-a",
      services: ["hosting", "workerserver"],
      manual: false,
      processes: [
        {
          service: "hosting",
          instanceId: "abc",
          goroutines: 120,
          heapBytes: 40 * 1024 ** 2,
          rssBytes: 90 * 1024 ** 2,
        },
      ],
      sample: {
        ts: 1754100000,
        target: "node-a",
        reachable: true,
        cpuPercent: 24.5,
        cpuValid: true,
        cpuCores: 4,
        memTotalBytes: 16 * 1024 ** 3,
        memAvailableBytes: 6 * 1024 ** 3,
        load1: 0.85,
        load5: 0.61,
        load15: 0.42,
        netReceiveBytes: 100,
        netTransmitBytes: 50,
        bootTime: Math.floor(Date.now() / 1000) - 90000,
        filesystems: [
          {
            mountpoint: "/",
            device: "/dev/sda1",
            fsType: "ext4",
            sizeBytes: 50 * 1024 ** 3,
            availBytes: 20 * 1024 ** 3,
          },
          {
            mountpoint: "/mnt/data",
            device: "/dev/sdb1",
            fsType: "xfs",
            sizeBytes: 1024 ** 4,
            availBytes: 400 * 1024 ** 3,
          },
        ],
      },
    },
  ],
};

describe("~/pages/admin/Health/Health.tsx", () => {
  let wrapper: RenderResult;

  const scope = (metrics: Metrics) =>
    nock(process.env.API_DOMAIN || "")
      .get("/admin/metrics")
      .reply(200, metrics);

  const targetsScope = (targets: string[] = []) =>
    nock(process.env.API_DOMAIN || "")
      .get("/admin/metrics/targets")
      .reply(200, { targets });

  const historyScope = (target: string, samples: Sample[]) =>
    nock(process.env.API_DOMAIN || "")
      .get(`/admin/metrics/history?target=${encodeURIComponent(target)}`)
      .reply(200, { target, samples });

  /** Two samples an hour apart, enough for the chart to have a line to draw. */
  const historySamples = (target: string): Sample[] => {
    const base = defaultMetrics.machines[0].sample!;

    return [
      { ...base, target, ts: 1754100000, cpuPercent: 10 },
      { ...base, target, ts: 1754103600, cpuPercent: 30 },
    ];
  };

  const createWrapper = async (metrics: Metrics = defaultMetrics) => {
    scope(metrics);
    targetsScope();

    metrics.machines.forEach(m => {
      historyScope(m.host, historySamples(m.host));
    });

    wrapper = render(<Health />);
    await waitFor(() => {
      expect(wrapper.container.innerHTML).not.toBe("");
    });
  };

  beforeEach(() => {
    nock.cleanAll();
  });

  it("renders a card per machine with its services", async () => {
    await createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("node-a")).toBeTruthy();
      expect(wrapper.getByText("hosting, workerserver")).toBeTruthy();
    });
  });

  // Every disk must be listed, not just the one Stormkit's volume sits on.
  it("lists every filesystem reported by the machine", async () => {
    await createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("/ (/dev/sda1)")).toBeTruthy();
      expect(wrapper.getByText("/mnt/data (/dev/sdb1)")).toBeTruthy();
    });
  });

  it("renders dependency health and the largest tables", async () => {
    await createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("PostgreSQL")).toBeTruthy();
      expect(wrapper.getByText("Redis")).toBeTruthy();
      expect(wrapper.getByText("public.access_logs")).toBeTruthy();
    });
  });

  it("shows the setup command when nothing is reporting", async () => {
    await createWrapper({ ...defaultMetrics, machines: [] });

    await waitFor(() => {
      expect(wrapper.getByText("No machines are reporting yet")).toBeTruthy();
    });
  });

  it("reports an unreachable machine instead of hiding it", async () => {
    await createWrapper({
      ...defaultMetrics,
      machines: [
        {
          host: "node-down",
          services: [],
          manual: true,
          sample: {
            ...defaultMetrics.machines[0].sample!,
            target: "node-down",
            reachable: false,
            error: "connection refused",
          },
        },
      ],
    });

    await waitFor(() => {
      expect(wrapper.getByText("Unreachable")).toBeTruthy();
      expect(wrapper.getByText("connection refused")).toBeTruthy();
    });
  });

  // Machine-wide numbers answer "is the box loaded"; these answer "is it
  // Stormkit's fault".
  it("shows what Stormkit's own processes use", async () => {
    await createWrapper();

    await waitFor(() => {
      expect(
        wrapper.getByText("Stormkit processes on this machine"),
      ).toBeTruthy();
      expect(wrapper.getByText("90 MB · 120 goroutines")).toBeTruthy();
    });
  });

  it("charts the retained history for a machine", async () => {
    await createWrapper();

    await waitFor(() => {
      expect(wrapper.getByTestId("area-chart")).toBeTruthy();
    });

    // The mocked chart serialises its data, so the points are assertable.
    expect(wrapper.getByTestId("area-chart").innerHTML).toContain('"cpu":10');
    expect(wrapper.getByTestId("area-chart").innerHTML).toContain('"cpu":30');
  });

  it("saves manual targets", async () => {
    await createWrapper();

    const updateScope = nock(process.env.API_DOMAIN || "")
      .put("/admin/metrics/targets", { targets: ["db-host:9100"] })
      .reply(200, { targets: ["db-host:9100"] });

    const input = await waitFor(() => wrapper.getByLabelText("Manual targets"));

    fireEvent.change(input, { target: { value: "db-host:9100" } });
    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(updateScope.isDone()).toBe(true);
    });
  });

  it("waits for the first sample rather than showing an error", async () => {
    await createWrapper({
      ...defaultMetrics,
      machines: [
        {
          host: "node-new",
          services: ["hosting"],
          manual: false,
          sample: null,
        },
      ],
    });

    await waitFor(() => {
      expect(
        wrapper.getByText(
          "Waiting for the first sample. This can take up to a minute.",
        ),
      ).toBeTruthy();
    });
  });
});

describe("~/pages/admin/Health/actions.ts", () => {
  it("formats bytes", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatBytes(50 * 1024 ** 3)).toBe("50 GB");
    expect(formatBytes(-1)).toBe("0 B");
  });

  it("formats uptime", () => {
    const now = Math.floor(Date.now() / 1000);

    expect(formatUptime(0)).toBe("-");
    expect(formatUptime(now - 30)).toBe("just booted");
    expect(formatUptime(now - 600)).toBe("10m");
    expect(formatUptime(now - 7200)).toBe("2h 0m");
    expect(formatUptime(now - 90000)).toBe("1d 1h");
  });
});
