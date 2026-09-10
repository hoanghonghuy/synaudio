import { expect, test } from '@playwright/test'

const chapter = {
  ID: 'chapter-1',
  StoryID: 'story-1',
  ChapterNumber: 1,
  Title: 'A deliberately long chapter title that must wrap without breaking the responsive layout',
  Status: 'READY',
  CurrentPlanRevisionID: 'plan-1',
}

async function mockAPI(page) {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname.replace('/api/v1', '')

    const json = (body, status = 200) => route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(body),
    })

    if (path === '/auth/me') return json({ id: 'admin-1', email: 'admin@example.test', roles: ['ADMIN'], status: 'ACTIVE', email_verified: true, mfa_enabled: false })
    if (path === '/me/favorites') return json({ stories: [] })
    if (path === '/stories/story-1/chapters') return json({ chapters: [chapter] })
    if (path === '/chapters/chapter-1/content') return json({ chapter_id: chapter.ID, title: chapter.Title, content: 'Readable chapter content remains available while audio state changes.' })
    if (path === '/chapters/chapter-1/audio-url') return json({ url: 'data:audio/mpeg;base64,' })
    if (path === '/admin/stories/story-1/chapters') return json({ chapters: [chapter] })
    if (path === '/admin/stories/story-1/workflow-settings') return json({ story_id: 'story-1', batch_generation_size: 1, creative_autonomy: 'ASSISTED', preferred_text_provider: 'mock', preferred_text_model: 'mock', preferred_tts_provider: 'mock', preferred_voice_id: 'voice-1' })
    if (path === '/admin/chapters/chapter-1/content') return json({ revisions: [] })
    if (path === '/admin/chapters/chapter-1/reviews') return json({ reviews: [] })
    if (path === '/admin/chapters/chapter-1/narration/latest') return json({ ID: 'nar-1', ChapterID: 'chapter-1', RevisionNo: 1, SourceContentRevisionID: 'cr-1', VoiceID: 'voice-1', Script: 'Hello.', Status: 'APPROVED' })
    if (path === '/admin/chapters/chapter-1/audio') return json({ ID: 'asset-1', ChapterID: 'chapter-1', VersionNo: 1, SourceNarrationRevisionID: 'nar-1', Status: 'READY', IsActive: true, Checksum: 'abc', StorageKey: 'private/key.mp3', DurationMs: 60000, SizeBytes: 1024 })
    if (path === '/admin/chapters/chapter-1/narration/nar-1/audio/latest-ready') return json({ ID: 'asset-1', ChapterID: 'chapter-1', VersionNo: 1, SourceNarrationRevisionID: 'nar-1', Status: 'READY', IsActive: true, Checksum: 'abc', StorageKey: 'private/key.mp3', DurationMs: 60000, SizeBytes: 1024 })
    if (path === '/admin/chapters/chapter-1/publish-readiness') return json({ ready: true, missing: [] })

    return json({ code: 'NOT_FOUND', message: 'fixture route not configured' }, 404)
  })
}

async function expectNoHorizontalOverflow(page) {
  const overflowing = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
  expect(overflowing).toBe(false)
}

for (const viewport of [
  { name: 'mobile', width: 390, height: 844 },
  { name: 'desktop', width: 1280, height: 900 },
]) {
  test(`listener reader is usable without horizontal overflow on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    await mockAPI(page)
    await page.goto('/stories/story-1/read')

    await expect(page.getByRole('heading', { name: chapter.Title })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Nghe chương này' })).toBeVisible()
    await expectNoHorizontalOverflow(page)

    await page.keyboard.press('Tab')
    const focusedTag = await page.evaluate(() => document.activeElement?.tagName)
    expect(focusedTag).not.toBe('BODY')
  })
}

for (const viewport of [
  { name: 'tablet', width: 820, height: 1000 },
  { name: 'desktop', width: 1280, height: 900 },
]) {
  test(`chapter production remains responsive and keyboard reachable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    await mockAPI(page)
    await page.goto('/admin/stories/story-1/production')

    await expect(page.getByRole('heading', { name: 'Chapter Production' })).toBeVisible()
    await expect(page.getByText(chapter.Title, { exact: true }).first()).toBeVisible()
    await expectNoHorizontalOverflow(page)

    await page.keyboard.press('Tab')
    const focusedTag = await page.evaluate(() => document.activeElement?.tagName)
    expect(focusedTag).not.toBe('BODY')
  })
}
