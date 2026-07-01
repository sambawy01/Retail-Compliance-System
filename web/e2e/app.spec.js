import { test, expect } from '@playwright/test'

const TEST_EMAIL = 'owner@easymart.com'
const TEST_PASSWORD = 'DemoPassword1234!'

test.describe('Authentication', () => {
  test('login page loads', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('h1')).toContainText('Watch Dog')
  })

  test('successful login redirects to dashboard', async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', TEST_EMAIL)
    await page.fill('input[type="password"]', TEST_PASSWORD)
    await page.click('button[type="submit"]')
    await page.waitForURL('**/', { timeout: 10000 })
    await expect(page.locator('h1')).toContainText('Dashboard')
  })

  test('wrong credentials show error', async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', 'wrong@test.com')
    await page.fill('input[type="password"]', 'wrongpassword')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Login failed')).toBeVisible({ timeout: 5000 })
  })

  test('unauthenticated access redirects to login', async ({ page }) => {
    await page.goto('/')
    await page.waitForURL('**/login', { timeout: 10000 })
    await expect(page).toHaveURL(/\/login/)
  })
})

test.describe('Navigation (authenticated)', () => {
  test.beforeEach(async ({ page }) => {
    // Login before each test
    await page.goto('/login')
    await page.fill('input[type="email"]', TEST_EMAIL)
    await page.fill('input[type="password"]', TEST_PASSWORD)
    await page.click('button[type="submit"]')
    await page.waitForURL('**/', { timeout: 10000 })
  })

  test('navigate to Cameras', async ({ page }) => {
    await page.click('a[href="/cameras"]')
    await page.waitForURL('**/cameras')
    await expect(page.locator('h1')).toContainText('Cameras')
  })

  test('navigate to Events', async ({ page }) => {
    await page.click('a[href="/events"]')
    await page.waitForURL('**/events')
    await expect(page.locator('h1')).toContainText('Events')
  })

  test('navigate to Face ID', async ({ page }) => {
    await page.click('a[href="/faceid"]')
    await page.waitForURL('**/faceid')
    await expect(page.locator('h1')).toContainText('Face ID')
  })

  test('navigate to Staff Performance', async ({ page }) => {
    await page.click('a[href="/staff"]')
    await page.waitForURL('**/staff')
    await expect(page.locator('h1')).toContainText('Staff')
  })

  test('navigate to Settings', async ({ page }) => {
    await page.click('a[href="/settings"]')
    await page.waitForURL('**/settings')
    await expect(page.locator('h1')).toContainText('Settings')
  })

  test('language toggle switches to Arabic', async ({ page }) => {
    // Click language toggle button
    await page.click('button:has-text("EN")')
    // Check that the page direction changes to RTL
    await expect(page.locator('html')).toHaveAttribute('dir', 'rtl')
  })

  test('logout returns to login', async ({ page }) => {
    await page.click('text=Logout')
    await page.waitForURL('**/login', { timeout: 10000 })
    await expect(page).toHaveURL(/\/login/)
  })
})

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', TEST_EMAIL)
    await page.fill('input[type="password"]', TEST_PASSWORD)
    await page.click('button[type="submit"]')
    await page.waitForURL('**/', { timeout: 10000 })
  })

  test('shows compliance score gauge', async ({ page }) => {
    await expect(page.locator('text=Compliance Score')).toBeVisible({ timeout: 10000 })
  })

  test('shows camera status section', async ({ page }) => {
    await expect(page.locator('text=Camera Status')).toBeVisible({ timeout: 10000 })
  })

  test('shows recent events feed', async ({ page }) => {
    await expect(page.locator('text=Recent Events')).toBeVisible({ timeout: 10000 })
  })
})

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', TEST_EMAIL)
    await page.fill('input[type="password"]', TEST_PASSWORD)
    await page.click('button[type="submit"]')
    await page.waitForURL('**/', { timeout: 10000 })
    await page.click('a[href="/settings"]')
    await page.waitForURL('**/settings')
  })

  test('shows notification rules section', async ({ page }) => {
    await expect(page.locator('text=Notification Rules')).toBeVisible({ timeout: 10000 })
  })

  test('shows organization settings', async ({ page }) => {
    await expect(page.locator('text=Organization Settings')).toBeVisible({ timeout: 10000 })
  })
})