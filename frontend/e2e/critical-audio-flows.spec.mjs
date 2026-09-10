import { expect, test } from '@playwright/test'

const chapter = {
  ID: 'chapter-1', StoryID: 'story-1', ChapterNumber: 1,
  Title: 'A deliberately long chapter title that must wrap without breaking the responsive layout',
  Status: 'READY', CurrentPlanRevisionID: 'plan-1',
}
const viewports = [
  { name: 'mobile', width: 390, height: 844 },
  { name: 'tablet', width: 820, height: 1000 },
  { name: 'desktop', width: 1280, height: 900 },
]
const progress = { UserID: 'admin-1', ChapterID: chapter.ID, PositionMs: 0, CompletedAt: '', LastAudioAssetID: '', LastPlaybackSessionID: '', Version: 0, RelistenStatus: 'NO_RELISTEN_NEEDED' }

async function mockAPI(page, options = {}) {
  let chapterListAttempts = 0
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname.replace('/api/v1', '')
    const method = route.request().method()
    const json = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
    if (path === '/auth/me') return json({ id: 'admin-1', email: 'admin@example.test', roles: ['ADMIN'], status: 'ACTIVE', email_verified: true, mfa_enabled: false })
    if (path === '/me/favorites') return json({ favorites: [] })
    if (path === '/me/progress/chapter-1' && method === 'GET') return json(progress)
    if (path === '/me/progress/chapter-1' && method === 'PUT') return json({ ...progress, PositionMs: 15000, Version: 1 })
    if (path === '/stories/story-1/chapters') {
      chapterListAttempts += 1
      if (options.delayChapterListMs) await new Promise((resolve) => setTimeout(resolve, options.delayChapterListMs))
      if (options.failChapterListOnce && chapterListAttempts === 1) return json({ code: 'TEMPORARY', message: 'temporary chapter list failure' }, 503)
      return json({ chapters: [chapter] })
    }
    if (path === '/chapters/chapter-1/content') return json({ chapter_id: chapter.ID, title: chapter.Title, content_text: 'Readable chapter content remains available while audio state changes.' })
    if (path === '/chapters/chapter-1/audio-url') {
      if (options.failAudio) return json({ code: 'AUDIO_UNAVAILABLE', message: 'fixture audio unavailable' }, 503)
      return json({ url: 'data:audio/mpeg;base64,' })
    }
    if (path === '/admin/stories/story-1/chapters') return json({ chapters: [chapter] })
    if (path === '/admin/stories/story-1/workflow-settings') return json({ story_id: 'story-1', batch_generation_size: 1, creative_autonomy: 'ASSISTED', preferred_text_provider: 'mock', preferred_text_model: 'mock', preferred_tts_provider: 'mock', preferred_voice_id: 'voice-1' })
    if (path === '/admin/chapters/chapter-1/content') return json({ revisions: [] })
    if (path === '/admin/chapters/chapter-1/reviews') return json({ reviews: [] })
    if (path === '/admin/chapters/chapter-1/narration/latest') return json({ ID: 'nar-1', ChapterID: 'chapter-1', RevisionNo: 1, SourceContentRevisionID: 'cr-1', VoiceID: 'voice-1', Script: 'Hello.', Status: 'APPROVED' })
    if (path === '/admin/chapters/chapter-1/audio') return json({ ID: 'asset-1', ChapterID: 'chapter-1', VersionNo: 1, SourceNarrationRevisionID: 'nar-1', Status: 'READY', IsActive: true, Checksum: 'abc', StorageKey: 'private/key.mp3', DurationMs: 60000, SizeBytes: 1024 })
    if (path === '/admin/chapters/chapter-1/narration/nar-1/audio/latest-ready') return json({ ID: 'asset-1', ChapterID: 'chapter-1', VersionNo: 1, SourceNarrationRevisionID: 'nar-1', Status: 'READY', IsActive: true, Checksum: 'abc', StorageKey: 'private/key.mp3', DurationMs: 60000, SizeBytes: 1024 })
    if (path === '/admin/chapters/chapter-1/publish-readiness') return json({ ready: true, missing: [] })
    return json({ code: 'NOT_FOUND', message: `fixture route not configured: ${method} ${path}` }, 404)
  })
}

async function expectNoHorizontalOverflow(page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)).toBe(false)
}
async function expectVisibleFocus(locator) {
  await expect(locator).toBeFocused()
  const s = await locator.evaluate((el) => { const x = getComputedStyle(el); return { outlineStyle: x.outlineStyle, outlineWidth: x.outlineWidth, boxShadow: x.boxShadow } })
  expect((s.outlineStyle !== 'none' && s.outlineWidth !== '0px') || s.boxShadow !== 'none').toBe(true)
}
async function tabTo(page, locator, maxTabs = 12) {
  await page.locator('body').click({ position: { x: 1, y: 1 } })
  await page.evaluate(() => document.activeElement instanceof HTMLElement && document.activeElement.blur())
  for (let i = 0; i < maxTabs; i += 1) {
    await page.keyboard.press('Tab')
    if (await locator.evaluate((el) => el === document.activeElement)) return
  }
  throw new Error('target was not reachable through keyboard Tab traversal')
}

for (const viewport of viewports) {
  test(`listener reader is responsive and keyboard reachable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height }); await mockAPI(page); await page.goto('/stories/story-1/read')
    await expect(page.getByRole('heading', { name: chapter.Title })).toBeVisible(); await expect(page.getByRole('heading', { name: 'Nghe chương này' })).toBeVisible(); await expect(page.getByText('Readable chapter content remains available while audio state changes.')).toBeVisible(); await expectNoHorizontalOverflow(page)
    const back = page.getByRole('link', { name: /Về chi tiết truyện/ }); const favorite = page.getByRole('button', { name: /Yêu thích/ }); const chapterButton = page.getByRole('button', { name: new RegExp(chapter.Title) })
    await tabTo(page, back); await expectVisibleFocus(back); await page.keyboard.press('Tab'); await expectVisibleFocus(favorite); await page.keyboard.press('Tab'); await expectVisibleFocus(chapterButton)
    await page.keyboard.press('Shift+Tab'); await expectVisibleFocus(favorite)
    await expect(page.locator('audio[controls]')).toBeVisible()
  })
}
for (const viewport of viewports) {
  test(`chapter production is responsive and keyboard reachable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height }); await mockAPI(page); await page.goto('/admin/stories/story-1/production')
    await expect(page.getByRole('heading', { name: 'Chapter Production' })).toBeVisible(); await expect(page.getByText(chapter.Title, { exact: true }).first()).toBeVisible(); await expectNoHorizontalOverflow(page)
    const back = page.getByRole('link', { name: /Story Planning Studio/ }); const chapterButton = page.getByRole('button', { name: new RegExp(chapter.Title) }).first()
    await tabTo(page, back); await expectVisibleFocus(back); await page.keyboard.press('Tab'); await expectVisibleFocus(chapterButton); await page.keyboard.press('Shift+Tab'); await expectVisibleFocus(back)
    await expect(page.getByRole('button', { name: /Start Generation|Refresh Generation State/ })).toBeVisible()
  })
}

test('listener exposes deterministic initial loading feedback', async ({ page }) => { await mockAPI(page, { delayChapterListMs: 1500 }); await page.goto('/stories/story-1/read'); await expect(page.getByText('Đang mở thư viện chương...')).toBeVisible(); await expect(page.getByRole('heading', { name: chapter.Title })).toBeVisible() })
test('listener exposes deterministic failure and retry feedback', async ({ page }) => { await mockAPI(page, { failChapterListOnce: true }); await page.goto('/stories/story-1/read'); await expect(page.getByRole('alert')).toContainText('Không thể tải các chương.'); await page.getByRole('button', { name: 'Thử lại' }).click(); await expect(page.getByRole('heading', { name: chapter.Title })).toBeVisible() })
test('listener keeps readable content available when audio URL fails', async ({ page }) => { await mockAPI(page, { failAudio: true }); await page.goto('/stories/story-1/read'); await expect(page.getByText('Audio tạm thời chưa sẵn sàng.')).toBeVisible(); await expect(page.getByRole('button', { name: 'Thử tải lại audio' })).toBeVisible(); await expect(page.getByText('Readable chapter content remains available while audio state changes.')).toBeVisible() })

test('current media boundary exposes visible play pause buffering error and recovery states', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/stories/story-1/read')
  const audio = page.locator('audio[controls]')
  await expect(audio).toBeVisible()

  const mediaValues = await audio.evaluate((el) => {
    el.playbackRate = 1.5
    el.currentTime = 15
    return { controls: el.controls, playbackRate: el.playbackRate, currentTime: el.currentTime }
  })
  expect(mediaValues).toEqual({ controls: true, playbackRate: 1.5, currentTime: 15 })

  await audio.dispatchEvent('play')
  await expect(page.getByText('Đang phát audio.')).toBeVisible()

  await audio.dispatchEvent('waiting')
  await expect(page.getByText('Audio đang tải thêm dữ liệu...')).toBeVisible()

  await audio.dispatchEvent('pause')
  await expect(page.getByText('Audio đang tạm dừng.')).toBeVisible()

  await audio.dispatchEvent('error')
  const playbackError = page.getByRole('alert')
  await expect(playbackError).toContainText('Audio gặp lỗi khi phát.')
  await expect(playbackError.getByRole('button', { name: 'Thử tải lại audio' })).toBeVisible()
  await expect(page.getByText('Readable chapter content remains available while audio state changes.')).toBeVisible()

  await playbackError.getByRole('button', { name: 'Thử tải lại audio' }).click()
  await expect(playbackError).toBeHidden()
  await expect(audio).toBeVisible()
})
