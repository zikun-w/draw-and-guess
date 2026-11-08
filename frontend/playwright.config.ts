import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 60000,
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: process.env.E2E_BASE_URL || "http://127.0.0.1:5173",
    channel: "msedge",
    launchOptions: {
      args: process.env.E2E_HOST_RULES
        ? [
            `--host-resolver-rules=${process.env.E2E_HOST_RULES}`,
            "--no-proxy-server",
          ]
        : [],
    },
    headless: true,
    viewport: { width: 1440, height: 1000 },
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
});
