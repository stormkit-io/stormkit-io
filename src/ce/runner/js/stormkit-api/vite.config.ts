import { defineConfig } from "vite";
import { resolve } from "node:path";

const config = defineConfig({
  ssr: {
    noExternal: ["@stormkit/serverless"],
  },
  build: {
    lib: {
      entry: resolve(__dirname, "api-entry.ts"),
      name: "stormkit-api",
      fileName: "stormkit-api",
      formats: ["es"],
    },
    outDir: "dist",
    minify: false,
    rollupOptions: {
      external: [
        "@google-cloud/functions-framework",
        "node:stream",
        "node:http",
        "node:path",
        "node:fs",
        "stream",
        "http",
        "path",
        "fs",
      ],
    },
  },
  resolve: {
    alias: {
      "~": resolve(__dirname, "src"),
    },
  },
});

export default config;
