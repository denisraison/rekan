import { defineConfig } from '@playwright/test';
const STORAGE_STATE = 'tests/.auth/operador.json';

export default defineConfig({
  testDir: 'tests',
  fullyParallel: true,
  webServer: {
    command: 'pnpm dev',
    port: 5173,
    reuseExistingServer: true,
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
    },
    {
      name: 'default',
      testIgnore: /auth\.setup\.ts/,
      use: { storageState: STORAGE_STATE },
      dependencies: ['setup'],
    },
  ],
  use: {
    baseURL: 'https://localhost:5173',
    ignoreHTTPSErrors: true,
  },
});
