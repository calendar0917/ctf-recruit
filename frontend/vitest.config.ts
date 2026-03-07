import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    exclude: [
      "node_modules/**",
      "node_modules.root-owned.bak/**",
      "e2e/**",
      "dist/**",
      "cypress/**",
      ".{idea,git,cache,output,temp}/**",
      "{karma,rollup,webpack,vite,vitest,jest,ava,babel,nyc,cypress,tsup,build}.config.*",
    ],
    coverage: {
      exclude: ["node_modules.root-owned.bak/**"],
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})
