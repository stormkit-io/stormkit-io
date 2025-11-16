import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "node:path";

export default defineConfig({
  // @ts-ignore
  plugins: [react()],
  test: {
    include: ["**/*.spec.tsx"],
    globals: true,
    silent: true,
    environment: "jsdom",
    setupFiles: ["src/testing/setup.tsx"],
    retry: 2,
    browser: {
      name: "Chrome",
      viewport: {
        width: 1920,
        height: 1080,
      },
    },
    env: {
      API_DOMAIN: "http://127.0.0.1:9876",
      NODE_ENV: "test",
    },
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html", "lcov"],
      reportsDirectory: "./coverage",
      all: true,
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/**/*.spec.tsx",
        "src/**/*.test.tsx",
        "src/testing/**",
        "src/**/__mocks__/**",
        "src/**/__tests__/**",
        "**/*.d.ts",
        "src/types/**",
        "**/node_modules/**",
      ],
      thresholds: {
        lines: 60,
        functions: 60,
        branches: 60,
        statements: 60,
      },
    },
  },
  resolve: {
    alias: {
      "~": path.join(__dirname, "src"),
    },
  },
});
