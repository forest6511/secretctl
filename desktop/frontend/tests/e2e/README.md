# E2E Tests for secretctl Desktop App

## Prerequisites

- Node.js 18+
- Playwright browsers installed
- Wails CLI installed

## Setup

```bash
# Install Playwright browsers (if not already done)
npx playwright install chromium
```

## Running Tests

Tests are fully automated. Playwright will:
1. Clean the vault directory (`/tmp/secretctl-e2e-test`)
2. Start the Wails dev server automatically
3. Run all tests
4. Shut down the server when done

```bash
cd desktop/frontend

# Run all E2E tests (auto-starts server)
npm run test:e2e

# Run tests with visible browser
npm run test:e2e:headed

# Run tests with Playwright UI
npm run test:e2e:ui

# Run specific test file
npx playwright test tests/e2e/auth.spec.ts
```

### Manual Server Mode

If you prefer to run the server manually (faster for development):

```bash
# Terminal 1: Start server
rm -rf /tmp/secretctl-e2e-test && mkdir -p /tmp/secretctl-e2e-test
cd desktop && SECRETCTL_VAULT_DIR=/tmp/secretctl-e2e-test wails dev

# Terminal 2: Run tests (will reuse existing server)
cd desktop/frontend && npm run test:e2e
```

## Test Structure

- `auth.spec.ts` - Authentication tests (SEC-001, SEC-002)
  - Vault creation validation
  - Password requirements
  - Vault unlock flow

- `secrets.spec.ts` - CRUD tests (CORE-001 to CORE-006)
  - Create secret
  - Read secret
  - Update secret
  - Delete secret
  - Search/filter secrets
  - Copy to clipboard

## Important Notes

1. **Fresh State Required**: Auth tests (SEC-001) require a fresh vault. Restart the dev server with a clean vault directory before running auth tests.

2. **State Persistence**: Wails keeps session state in memory. Deleting vault files won't reset the running app - you must restart the server.

3. **Test Isolation**: Tests are designed to handle various app states (create mode, unlock mode, secrets page) in beforeEach hooks.

## Troubleshooting

### Tests fail with "ERR_CONNECTION_REFUSED"
The Wails dev server is not running. Start it using the instructions above.

### Tests fail expecting "Create Vault" but see "Secrets" page
The app is already logged in from a previous session. Restart the dev server with a fresh vault directory.

### Browser doesn't appear
Playwright runs in headless mode by default. Use `npm run test:e2e:headed` to see the browser.
