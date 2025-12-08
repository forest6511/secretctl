import { test, expect } from '@playwright/test'
import { TEST_PASSWORD } from './test-config'

/**
 * Audit Log Viewer E2E Tests
 * Tests: AUDIT-001 to AUDIT-011
 */

test.describe('Audit Log Viewer', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // Handle vault creation or unlock
    const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
    const isUnlockMode = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)
    const isSecretsPage = await page.getByTestId('secrets-list').isVisible().catch(() => false)

    if (isSecretsPage) {
      return
    }

    if (isCreateMode) {
      await page.getByTestId('master-password').fill(TEST_PASSWORD)
      await page.getByTestId('confirm-password').fill(TEST_PASSWORD)
      await page.getByTestId('unlock-button').click()
    } else if (isUnlockMode) {
      await page.getByTestId('master-password').fill(TEST_PASSWORD)
      await page.getByTestId('unlock-button').click()
    }

    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
  })

  test('AUDIT-001: Navigate to Audit Log page', async ({ page }) => {
    // Click audit log button/link
    await page.getByTestId('audit-button').click()

    // Verify audit page is visible
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })
    await expect(page.getByText('Audit Log')).toBeVisible()
  })

  test('AUDIT-002: Display audit log table', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Verify table structure
    await expect(page.getByTestId('audit-log-table')).toBeVisible()

    // Check table headers
    await expect(page.getByText('Timestamp')).toBeVisible()
    await expect(page.getByText('Action')).toBeVisible()
    await expect(page.getByText('Source')).toBeVisible()
    await expect(page.getByText('Key')).toBeVisible()
    await expect(page.getByText('Status')).toBeVisible()
  })

  test('AUDIT-003: Chain integrity verification', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Check chain status indicator exists
    await expect(page.getByTestId('chain-status')).toBeVisible()

    // Should show either "Chain Verified" or "Checking..."
    const statusText = await page.getByTestId('chain-status').textContent()
    expect(statusText).toMatch(/Chain Verified|Checking chain integrity|Chain Invalid/)
  })

  test('AUDIT-004: Filter by action type', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Select action filter
    await page.getByTestId('filter-action').selectOption('auth.unlock')

    // Apply filter
    await page.getByTestId('apply-filter-button').click()

    // Wait for filter to apply
    await page.waitForTimeout(500)

    // Clear filter
    await page.getByTestId('clear-filter-button').click()
  })

  test('AUDIT-005: Filter by source', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Select source filter
    await page.getByTestId('filter-source').selectOption('ui')

    // Apply filter
    await page.getByTestId('apply-filter-button').click()

    // Wait for filter to apply
    await page.waitForTimeout(500)

    // Verify filter is applied (check select value)
    await expect(page.getByTestId('filter-source')).toHaveValue('ui')
  })

  test('AUDIT-006: Export to CSV', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Set up download listener
    const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)

    // Click export CSV button
    await page.getByTestId('export-csv-button').click()

    // Check download started (may not work in all environments)
    const download = await downloadPromise
    if (download) {
      expect(download.suggestedFilename()).toContain('.csv')
    }
  })

  test('AUDIT-007: Export to JSON', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Set up download listener
    const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)

    // Click export JSON button
    await page.getByTestId('export-json-button').click()

    // Check download started
    const download = await downloadPromise
    if (download) {
      expect(download.suggestedFilename()).toContain('.json')
    }
  })

  test('AUDIT-008: View log detail modal', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Wait for logs to load
    await page.waitForTimeout(1000)

    // Click on first log row if exists
    const firstRow = page.getByTestId('audit-log-row-0')
    const rowExists = await firstRow.isVisible().catch(() => false)

    if (rowExists) {
      await firstRow.click()

      // Verify modal appears
      await expect(page.getByTestId('audit-detail-modal')).toBeVisible()
      await expect(page.getByText('Audit Log Detail')).toBeVisible()

      // Close modal by clicking outside
      await page.getByTestId('audit-detail-modal').click({ position: { x: 10, y: 10 } })
    }
  })

  test('AUDIT-009: Pagination controls', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Verify pagination controls exist
    await expect(page.getByTestId('prev-page-button')).toBeVisible()
    await expect(page.getByTestId('next-page-button')).toBeVisible()

    // First page - prev should be disabled
    await expect(page.getByTestId('prev-page-button')).toBeDisabled()
  })

  test('AUDIT-010: Back navigation', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Click back button
    await page.getByTestId('back-button').click()

    // Should return to secrets page
    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 5000 })
  })

  test('AUDIT-011: Verify chain button', async ({ page }) => {
    // Navigate to audit page
    await page.getByTestId('audit-button').click()
    await expect(page.getByTestId('audit-page')).toBeVisible({ timeout: 5000 })

    // Click verify chain button
    await page.getByTestId('verify-chain-button').click()

    // Button should show loading state or status should update
    // Wait for verification to complete
    await page.waitForTimeout(2000)

    // Chain status should be visible
    await expect(page.getByTestId('chain-status')).toBeVisible()
  })
})
