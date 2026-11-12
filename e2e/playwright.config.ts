import { defineConfig } from '@playwright/test'

// Runs against a server already started on BASE_URL and seeded with cmd/demoseed (see CLAUDE.md).
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  projects: [
    { name: 'setup', testMatch: /auth\.setup\.ts/ },
    { name: 'flows', testMatch: /(flows|a11y|mobile|oauth|mcp-sdk)\.spec\.ts/, dependencies: ['setup'], use: { storageState: '.auth/state.json' } },
  ],
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:8080',
    locale: 'en-GB',
    // The server cuts its days in its reporting timezone (APP_TIMEZONE, Europe/Paris unless set) and the period bar
    // cuts "today" with the browser clock: the two must agree, or between midnight in Paris and midnight in UTC a
    // record created by a test is dated tomorrow and falls outside the period the page asks for.
    timezoneId: process.env.APP_TIMEZONE ?? 'Europe/Paris',
    trace: 'retain-on-failure',
  },
})
