import { test, expect } from '@playwright/test'

/**
 * Secrets CRUD E2E Tests
 * Tests: CORE-001 to CORE-005 (P0 Critical)
 */

test.describe('Secrets Management', () => {
  const TEST_PASSWORD = 'SecurePassword123!'
  const TEST_SECRET = {
    key: 'test/api-key',
    value: 'sk-test-12345',
    url: 'https://api.example.com',
    tags: 'test, api',
    notes: 'Test API key for E2E testing',
  }

  test.beforeEach(async ({ page }) => {
    await page.goto('/')

    // Wait for page to load
    await page.waitForLoadState('networkidle')

    // Handle vault creation or unlock
    const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
    const isUnlockMode = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)
    const isSecretsPage = await page.getByTestId('secrets-list').isVisible().catch(() => false)

    if (isSecretsPage) {
      // Already on secrets page, nothing to do
      return
    }

    if (isCreateMode) {
      // Create new vault
      await page.getByTestId('master-password').fill(TEST_PASSWORD)
      await page.getByTestId('confirm-password').fill(TEST_PASSWORD)
      await page.getByTestId('unlock-button').click()
    } else if (isUnlockMode) {
      // Unlock existing vault
      await page.getByTestId('master-password').fill(TEST_PASSWORD)
      await page.getByTestId('unlock-button').click()
    }

    // Wait for secrets page to load
    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
  })

  test('CORE-001: Create secret with all fields', async ({ page }) => {
    // Click add secret button
    await page.getByTestId('add-secret-button').click()

    // Fill form
    await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
    await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
    await page.getByTestId('secret-url-input').fill(TEST_SECRET.url)
    await page.getByTestId('secret-tags-input').fill(TEST_SECRET.tags)
    await page.getByTestId('secret-notes-input').fill(TEST_SECRET.notes)

    // Save
    await page.getByTestId('save-secret-button').click()

    // Verify secret appears in list
    await expect(page.getByText(TEST_SECRET.key)).toBeVisible()
  })

  test('CORE-002: Read secret value', async ({ page }) => {
    // First create a secret if it doesn't exist
    const secretExists = await page.getByText(TEST_SECRET.key).isVisible().catch(() => false)

    if (!secretExists) {
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
      await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
      await page.getByTestId('save-secret-button').click()
      await expect(page.getByText(TEST_SECRET.key)).toBeVisible()
    }

    // Click on the secret to view details
    await page.getByText(TEST_SECRET.key).click()

    // Value should be hidden by default (password field)
    const valueDisplay = page.getByTestId('secret-value-display')
    await expect(valueDisplay).toBeVisible()

    // Toggle visibility
    await page.getByTestId('toggle-value-visibility').click()

    // Value should now be visible
    await expect(valueDisplay).toHaveValue(TEST_SECRET.value)
  })

  test('CORE-003: Update secret', async ({ page }) => {
    const updatedValue = 'sk-updated-67890'

    // Ensure secret exists
    const secretExists = await page.getByText(TEST_SECRET.key).isVisible().catch(() => false)

    if (!secretExists) {
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
      await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
      await page.getByTestId('save-secret-button').click()
    }

    // Select secret
    await page.getByText(TEST_SECRET.key).click()

    // Click edit
    await page.getByTestId('edit-secret-button').click()

    // Update value
    await page.getByTestId('secret-value-input').clear()
    await page.getByTestId('secret-value-input').fill(updatedValue)

    // Save
    await page.getByTestId('save-secret-button').click()

    // Verify update
    await page.getByText(TEST_SECRET.key).click()
    await page.getByTestId('toggle-value-visibility').click()
    await expect(page.getByTestId('secret-value-display')).toHaveValue(updatedValue)
  })

  test('CORE-004: Delete secret', async ({ page }) => {
    const deleteKey = 'test/to-delete'

    // Create a secret to delete
    await page.getByTestId('add-secret-button').click()
    await page.getByTestId('secret-key-input').fill(deleteKey)
    await page.getByTestId('secret-value-input').fill('delete-me')
    await page.getByTestId('save-secret-button').click()

    await expect(page.getByText(deleteKey)).toBeVisible()

    // Select and delete
    await page.getByText(deleteKey).click()

    // Handle confirmation dialog
    page.on('dialog', dialog => dialog.accept())
    await page.getByTestId('delete-secret-button').click()

    // Verify deletion
    await expect(page.getByText(deleteKey)).not.toBeVisible()
  })

  test('CORE-005: Secret list display and search', async ({ page }) => {
    // Create multiple secrets
    const secrets = ['search/alpha', 'search/beta', 'other/gamma']

    for (const key of secrets) {
      const exists = await page.getByText(key).isVisible().catch(() => false)
      if (!exists) {
        await page.getByTestId('add-secret-button').click()
        await page.getByTestId('secret-key-input').fill(key)
        await page.getByTestId('secret-value-input').fill(`value-${key}`)
        await page.getByTestId('save-secret-button').click()
        await expect(page.getByText(key)).toBeVisible()
      }
    }

    // Test search functionality
    await page.getByTestId('search-secrets').fill('search')

    // Should show filtered results
    await expect(page.getByText('search/alpha')).toBeVisible()
    await expect(page.getByText('search/beta')).toBeVisible()
    await expect(page.getByText('other/gamma')).not.toBeVisible()

    // Clear search
    await page.getByTestId('search-secrets').clear()

    // All secrets should be visible again
    await expect(page.getByText('other/gamma')).toBeVisible()
  })

  test('CORE-006: Copy secret to clipboard', async ({ page }) => {
    // Ensure secret exists
    const secretExists = await page.getByText(TEST_SECRET.key).isVisible().catch(() => false)

    if (!secretExists) {
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
      await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
      await page.getByTestId('save-secret-button').click()
    }

    // Select secret
    await page.getByText(TEST_SECRET.key).click()

    // Click copy button
    await page.getByTestId('copy-secret-button').click()

    // Should show copy feedback
    await expect(page.getByText('Copied!')).toBeVisible()
  })
})
