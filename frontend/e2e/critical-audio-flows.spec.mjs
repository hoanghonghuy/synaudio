import { expect, test } from '@playwright/test'

const chapter = {
  ID: 'chapter-1',
  StoryID: 'story-1',
  ChapterNumber: 1,
  Title: 'A deliberately long chapter title that must wrap without breaking the responsive layout',
  Status: 'READY',
  CurrentPlanRevisionID: 'plan-1',
}

const viewports = [
  { name: 'mobile', width: 390, height: 844 },
  { name: 'tablet', width: 820, height: 1000 },
  { name: 'desktop', width: 1280, height: 900 },
]

async function mockAPI(page, options = {}) {
  let chapterListAttempts = 0
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
    if (path === '/me/progress/chapter-1') return json({ ChapterID: chapter.ID, PositionMs: 0, Completed: false, RelistenStatus: 'NO_RELISTEN_NEEDED' })

    if (path === '/stories/story-1/chapters') {
      chapterListAttempts += 1
      if (options.failChapterListOnce && chapterListAttempts === 1) {
        return json({ code: 'TEMPORARY', message: 'temporary chapter list failure' }, 503)
      }
      return json({ chapters: [chapter] })
    }
    if (path === '/chapters/chapter-1/content') {
      if (options.delayContentMs) await new Promise((resolve) => setTimeout(resolve, options.delayContentMs))
      return json({ chapter_id: chapter.ID, title: chapter.Title, content_text: 'Readable chapter content remains available while audio state changes.' })
    }
    if (path === '/chapters/chapter-1/audio-url') {
      if (options.failAudio) return json({ code: 'AUDIO_UNAVAILABLE', message: 'fixture audio unavailable' }, 503)
      return json({ url: 'data:audio/mpeg;base64,' })
    }
    if (path === '/me/progress/chapter-1' && route.request().method() !== 'GET') return json({ ChapterID: chapter.ID, PositionMs: 15000, Completed: false, RelistenStatus: 'NO_RELISTEN_NEEDED' })

    if (path === '/admin/stories/story-1/chapters') return json({ chapters: [chapter] })
    if (path === '/admin/stories/story-1/workflow-settings') return json({ story_id: 'story-1', batch_generation_size: 1, creative_autonomy: 'ASSISTED', preferred_text_provider: 'mock', preferred_text_model: 'mock', preferred_tts_provider: 'mock', preferred_voice_id: 'voice-1' })
    if (path === '/admin/chapters/chapter-1/content') return json({ revisions: [] })
    if (path === '/admin/chapters/chapter-1/reviews') return json({ reviews: [] })
    if (path === '/admin/chapters/chapter-1/narration/latest') return json({ ID: 'nar-1', ChapterID: 'chapter-1', RevisionNo: 1, SourceContentRevisionID: 'cr-1', VoiceID: 'voice-1', Script: 'Hello.', Status: 'APPROVED' })
    if (path === '/admin/chapters/chapter-1/audio') return json({ ID: 'asset-1', ChapterID: 'chapter-1', VersionNo: 1, SourceNarrationRevisionID: 'nar-1', Status: 'READY', IsActive: true, Checksum: 'abc', StorageKey: 'private/key.mp3', DurationMs: 60000, SizeBytes: 1024 })
    if (path === '/admin/chapters/chapter-1/narration/nar-1/audio/latest-ready') return json({ ID: 'asset-1', ChapterID: 'chapter-1', VersionNo: 1, SourceNarrationRevisionID: 'nar-1', Status: 'READY', IsActive: true, Checksum: 'abc', StorageKey: 'private/key.mp3', DurationMs: 60000, SizeBytes: 1024 })
    if (path === '/admin/chapters/chapter-1/publish-readiness') return json({ ready: true, missing: [] })

    return json({ code: 'NOT_FOUND', message: `fixture route not configured: ${route.request().method()} ${path}` }, 404)
  })
}

async function expectNoHorizontalOverflow(page) {
  const overflowing = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
  expect(overflowing).toBe(false)
}

async function expectVisibleKeyboardFocus(page, locator) {
  await locator.focus()
  await expect(locator).toBeFocused()
  const focusStyle = await locator.evaluate((element) => {
    const style = getComputedStyle(element)
    return {
      outlineStyle: style.outlineStyle,
      outlineWidth: style.outlineWidth,
      boxShadow: style.boxShadow,
    }
  })
  const hasVisibleFocus = focusStyle.outlineStyle !== 'none' && focusStyle.outlineWidth !== '0px'
    || focusStyle.boxShadow !== 'none'
  expect(hasVisibleFocus).toBe(true)
}

for (const viewport of viewports) {
  test(`listener reader is responsive and keyboard reachable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    await mockAPI(page)
    await page.goto('/stories/story-1/read')

    await expect(page.getByRole('heading', { name: chapter.Title })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Nghe chương này' })).toBeVisible()
    await expect(page.getByText('Readable chapter content remains available while audio state changes.')).toBeVisible()
    await expectNoHorizontalOverflow(page)

    const favorite = page.getByRole('button', { name: /Yêu thích/ })
    await expectVisibleKeyboardFocus(page, favorite)
    await expect(page.locator('audio[controls]')).toBeVisible()
  })
}

for (const viewport of viewports) {
  test(`chapter production is responsive and keyboard reachable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    await mockAPI(page)
    await page.goto('/admin/stories/story-1/production')

    await expect(page.getByRole('heading', { name: 'Chapter Production' })).toBeVisible()
    await expect(page.getByText(chapter.Title, { exact: true }).first()).toBeVisible()
    await expectNoHorizontalOverflow(page)

    const chapterButton = page.getByRole('button', { name: new RegExp(chapter.Title) }).first()
    await expectVisibleKeyboardFocus(page, chapterButton)
    await expect(page.getByRole('button', { name: /Start Generation|Refresh Generation State/ })).toBeVisible()
  })
}

test('listener exposes deterministic loading, failure and retry feedback', async ({ page }) => {
  await mockAPI(page, { failChapterListOnce: true })
  await page.goto('/stories/story-1/read')

  await expect(page.getByRole('alert')).toContainText('Không thể tải các chương.')
  await page.getByRole('button', { name: 'Thử lại' }).click()
  await expect(page.getByRole('heading', { name: chapter.Title })).toBeVisible()
})

test('listener keeps readable content available when audio URL fails', async ({ page }) => {
  await mockAPI(page, { failAudio: true, delayContentMs: 100 })
  await page.goto('/stories/story-1/read')

  await expect(page.getByRole('status')).toContainText(/Đang tải nội dung|Đang chuẩn bị audio/)
  await expect(page.getByText('Audio tạm thời chưa sẵn sàng.')).toBeVisible()
  await expect(page.getByText('Readable chapter content remains available while audio state changes.')).toBeVisible()
})

test('native media boundary supports deterministic seek and rate state without remote audio', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/stories/story-1/read')
  const audio = page.locator('audio[controls]')
  await expect(audio).toBeVisible()

  const state = await audio.evaluate((element) => {
    element.playbackRate = 1.5
    element.currentTime = 15
    element.dispatchEvent(new Event('seeked'))
    element.dispatchEvent(new Event('pause'))
    return { playbackRate: element.playbackRate, currentTime: element.currentTime, controls: element.controls }
  })
  expect(state.controls).toBe(true)
  expect(state.playbackRate).toBe(1.5)
  expect(state.currentTime).toBe(15)
})
