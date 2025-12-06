import { execSync } from 'child_process'
import * as fs from 'fs'
import * as path from 'path'

const VAULT_DIR = '/tmp/secretctl-e2e-test'

/**
 * Global setup for E2E tests
 * Cleans vault directory to ensure fresh state for each test run
 */
export default async function globalSetup() {
  console.log('🧹 Cleaning vault directory:', VAULT_DIR)

  // Remove existing vault directory
  if (fs.existsSync(VAULT_DIR)) {
    fs.rmSync(VAULT_DIR, { recursive: true, force: true })
  }

  // Create fresh directory
  fs.mkdirSync(VAULT_DIR, { recursive: true, mode: 0o700 })

  console.log('✅ Vault directory ready')
}
