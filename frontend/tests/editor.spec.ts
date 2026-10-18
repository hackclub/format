import { afterAll, afterEach, beforeAll, beforeEach, expect, test } from 'bun:test'
import { chromium, type Browser, type BrowserContext, type Page } from 'playwright'

let browser: Browser
let context: BrowserContext
let page: Page
let server: ReturnType<typeof Bun.spawn>

beforeAll(async () => {
  server = Bun.spawn(['bun', Bun.resolveSync('next/dist/bin/next', import.meta.dir), 'start', '--hostname', '127.0.0.1', '--port', '3100'], {
    stdout: 'ignore', stderr: 'inherit',
  })
  const deadline = Date.now() + 30_000
  while (true) {
    if (server.exitCode !== null) throw new Error('Test server exited before becoming ready')
    try {
      if ((await fetch('http://127.0.0.1:3100')).ok) break
    } catch {}
    if (Date.now() >= deadline) throw new Error('Test server did not become ready')
    await Bun.sleep(100)
  }
  browser = await chromium.launch()
}, 40_000)

beforeEach(async () => {
  context = await browser.newContext({
    baseURL: 'http://127.0.0.1:3100', permissions: ['clipboard-read', 'clipboard-write'],
  })
  page = await context.newPage()
})

afterEach(async () => { await context?.close() })
afterAll(async () => {
  await browser?.close()
  server?.kill()
  if (server) await server.exited
})

test('signed-out users see the styled sign-in screen', async () => {
  await page.route('**/api/auth/me', route => route.fulfill({ status: 401 }))
  await page.goto('/')
  const login = page.getByRole('button', { name: 'Sign in with Google' })
  await login.waitFor({ state: 'visible' })
  expect(await login.evaluate(element => getComputedStyle(element).backgroundColor)).toBe('rgb(236, 55, 80)')
  expect(await page.locator('[contenteditable]').count()).toBe(0)
})

test('editor pastes rich HTML, formats selections, creates lists and copies HTML', async () => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => {
    if (message.type() === 'error') errors.push(message.text())
  })
  await page.route('**/api/auth/me', route => route.fulfill({ json: {
    sub: 'test-user', email: 'test@example.test', name: 'Test User', picture: '', hd: 'example.test',
  } }))
  await page.route('**/api/config', route => route.fulfill({ json: { cdnBaseUrl: 'https://cdn.example.test' } }))
  await page.route('https://cdn.example.test/**', route => route.fulfill({
    contentType: 'image/svg+xml', body: '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"/>',
  }))
  let exportedHTML = ''
  await page.route('**/api/html/transform', async route => {
    exportedHTML = route.request().postDataJSON().html
    await route.fulfill({ json: { html: exportedHTML, messages: [], stats: {
      images_processed: 1, images_rehosted: 0, styles_removed: 0, scripts_removed: 0,
    } } })
  })
  await page.goto('/')
  const editor = page.locator('[contenteditable="true"]')
  await editor.waitFor({ state: 'visible' })
  await editor.fill('Upgrade regression')
  await editor.press('ControlOrMeta+a')
  await page.getByTitle('Bold (Cmd+B)').click()
  await editor.locator('.editor-text-bold', { hasText: 'Upgrade regression' }).waitFor()
  expect(await page.getByTitle('Bold (Cmd+B)').getAttribute('class')).toContain('bg-blue-100')
  await page.getByTitle('Bullet List').click()
  await editor.locator('ul li', { hasText: 'Upgrade regression' }).waitFor()
  await editor.press('ControlOrMeta+a')
  await editor.evaluate(element => {
    const clipboardData = new DataTransfer()
    clipboardData.setData('text/html', '<p><b>Pasted bold</b> <i>italic</i> <a href="https://example.test">link</a></p><p><img src="https://cdn.example.test/image.svg" alt="Fixture image" width="20" height="20"></p>')
    clipboardData.setData('text/plain', 'Pasted bold italic link')
    element.dispatchEvent(new ClipboardEvent('paste', { clipboardData, bubbles: true, cancelable: true }))
  })
  await editor.getByText('Pasted bold').waitFor()
  await editor.locator('img[alt="Fixture image"]').first().waitFor()
  await page.getByRole('button', { name: '📋 Copy For Gmail' }).click()
  await page.getByRole('button', { name: '✅ Copied!' }).waitFor()
  expect(exportedHTML).toContain('Pasted bold')
  expect(exportedHTML).toContain('https://example.test')
  expect(exportedHTML).toContain('https://cdn.example.test/image.svg')
  const clipboardHTML = await page.evaluate(async () => {
    const items = await navigator.clipboard.read()
    const html = await items[0].getType('text/html')
    return html.text()
  })
  expect(clipboardHTML).toContain('Pasted bold')
  expect(errors).toEqual([])
})
