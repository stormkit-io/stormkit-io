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
  },
  resolve: {
    alias: {
      "~": path.join(__dirname, "src"),
    },
  },
});
