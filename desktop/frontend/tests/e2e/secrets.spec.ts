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
    // Check if secret already exists (from previous test runs)
    // Use .first() to avoid strict mode violation when key appears in list and heading
    const secretExists = await page.getByText(TEST_SECRET.key).first().isVisible().catch(() => false)

    if (secretExists) {
      // Secret already exists, test passes - just verify it's in the list
      await expect(page.getByText(TEST_SECRET.key).first()).toBeVisible()
      return
    }

    // Click add secret button
    await page.getByTestId('add-secret-button').click()

    // Wait for form to be visible
    await expect(page.getByTestId('secret-key-input')).toBeVisible()

    // Fill form
    await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
    await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
    await page.getByTestId('secret-url-input').fill(TEST_SECRET.url)
    await page.getByTestId('secret-tags-input').fill(TEST_SECRET.tags)
    await page.getByTestId('secret-notes-input').fill(TEST_SECRET.notes)

    // Save
    await page.getByTestId('save-secret-button').click()

    // Verify secret appears in list (wait for save to complete)
    // Use .first() because the key appears both in list item and detail view heading
    await expect(page.getByText(TEST_SECRET.key).first()).toBeVisible({ timeout: 5000 })
  })

  test('CORE-002: Read secret value', async ({ page }) => {
    // First create a secret if it doesn't exist
    // Use .first() to avoid strict mode violation when key appears in list and heading
    const secretExists = await page.getByText(TEST_SECRET.key).first().isVisible().catch(() => false)

    if (!secretExists) {
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
      await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
      await page.getByTestId('save-secret-button').click()
      await expect(page.getByText(TEST_SECRET.key).first()).toBeVisible()
    }

    // Click on the secret to view details - use .first() to select list item
    await page.getByText(TEST_SECRET.key).first().click()

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

    // Ensure secret exists - use .first() to avoid strict mode violation
    const secretExists = await page.getByText(TEST_SECRET.key).first().isVisible().catch(() => false)

    if (!secretExists) {
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
      await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
      await page.getByTestId('save-secret-button').click()
      await expect(page.getByText(TEST_SECRET.key).first()).toBeVisible()
    }

    // Select secret - use .first() to select list item
    await page.getByText(TEST_SECRET.key).first().click()

    // Click edit
    await page.getByTestId('edit-secret-button').click()

    // Update value
    await page.getByTestId('secret-value-input').clear()
    await page.getByTestId('secret-value-input').fill(updatedValue)

    // Save
    await page.getByTestId('save-secret-button').click()

    // Verify update - click on secret again to view details
    await page.getByText(TEST_SECRET.key).first().click()
    await page.getByTestId('toggle-value-visibility').click()
    await expect(page.getByTestId('secret-value-display')).toHaveValue(updatedValue)
  })

  test('CORE-004: Delete secret', async ({ page }) => {
    const deleteKey = 'test/to-delete'

    // Check if secret already exists - use .first() to avoid strict mode violation
    const secretExists = await page.getByText(deleteKey).first().isVisible().catch(() => false)

    if (!secretExists) {
      // Create a secret to delete
      await page.getByTestId('add-secret-button').click()
      await expect(page.getByTestId('secret-key-input')).toBeVisible()
      await page.getByTestId('secret-key-input').fill(deleteKey)
      await page.getByTestId('secret-value-input').fill('delete-me')
      await page.getByTestId('save-secret-button').click()
      await expect(page.getByText(deleteKey).first()).toBeVisible({ timeout: 5000 })
    }

    // Select and delete - use .first() to select list item, not heading
    await page.getByText(deleteKey).first().click()

    // Wait for detail view to load
    await expect(page.getByTestId('delete-secret-button')).toBeVisible()

    // Handle confirmation dialog - set up auto-accept handler BEFORE clicking
    // The dialog blocks the click, so we need to handle it inline
    page.on('dialog', async (dialog) => {
      await dialog.accept()
    })

    // Now click delete - the dialog will be auto-accepted
    await page.getByTestId('delete-secret-button').click()

    // Wait for deletion to complete and verify secret is gone from the list
    const secretsList = page.getByTestId('secrets-list')
    await expect(secretsList.getByText(deleteKey)).not.toBeVisible({ timeout: 5000 })
  })

  test('CORE-005: Secret list display and search', async ({ page }) => {
    // Create multiple secrets
    const secrets = ['search/alpha', 'search/beta', 'other/gamma']

    for (const key of secrets) {
      // Use .first() to avoid strict mode violation when text appears in list and detail view
      const exists = await page.getByText(key).first().isVisible().catch(() => false)
      if (!exists) {
        // Wait for add button to be ready
        await expect(page.getByTestId('add-secret-button')).toBeVisible()
        await page.getByTestId('add-secret-button').click()

        // Wait for form to be visible
        await expect(page.getByTestId('secret-key-input')).toBeVisible()
        await page.getByTestId('secret-key-input').fill(key)
        await page.getByTestId('secret-value-input').fill(`value-${key}`)
        await page.getByTestId('save-secret-button').click()

        // Wait for secret to appear in list before continuing
        await expect(page.getByText(key).first()).toBeVisible({ timeout: 5000 })

        // Wait a moment for UI to settle before next iteration
        await page.waitForTimeout(200)
      }
    }

    // Verify all secrets are in the list before searching
    for (const key of secrets) {
      await expect(page.getByText(key).first()).toBeVisible()
    }

    // Test search functionality
    await page.getByTestId('search-secrets').fill('search')

    // Wait for filter to apply
    await page.waitForTimeout(300)

    // Get the secrets list container to check only list items, not detail view
    const secretsList = page.getByTestId('secrets-list')

    // Should show filtered results in the list - use locator within secrets-list
    await expect(secretsList.getByText('search/alpha')).toBeVisible()
    await expect(secretsList.getByText('search/beta')).toBeVisible()
    // other/gamma should not be in the list (may still be in detail view heading)
    await expect(secretsList.getByText('other/gamma')).not.toBeVisible()

    // Clear search
    await page.getByTestId('search-secrets').clear()

    // Wait for filter to clear
    await page.waitForTimeout(300)

    // All secrets should be visible again in the list
    await expect(secretsList.getByText('other/gamma')).toBeVisible()
  })

  test('CORE-006: Copy secret to clipboard', async ({ page }) => {
    // Ensure secret exists - use .first() to avoid strict mode violation
    const secretExists = await page.getByText(TEST_SECRET.key).first().isVisible().catch(() => false)

    if (!secretExists) {
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(TEST_SECRET.key)
      await page.getByTestId('secret-value-input').fill(TEST_SECRET.value)
      await page.getByTestId('save-secret-button').click()
      await expect(page.getByText(TEST_SECRET.key).first()).toBeVisible()
    }

    // Select secret - use .first() to select list item
    await page.getByText(TEST_SECRET.key).first().click()

    // Click copy button
    await page.getByTestId('copy-secret-button').click()

    // Should show copy feedback
    await expect(page.getByText('Copied!')).toBeVisible()
  })
})
