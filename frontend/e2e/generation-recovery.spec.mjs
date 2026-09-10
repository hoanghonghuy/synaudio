import { expect, test } from '@playwright/test'

const chapter = {
  ID: 'chapter-1', StoryID: 'story-1', ChapterNumber: 1,
  Title: 'Generation recovery chapter', Status: 'DRAFT', CurrentPlanRevisionID: 'plan-1',
}

const viewports = [
  { name: 'mobile', width: 390, height: 844 },
  { name: 'tablet', width: 820, height: 1000 },
  { name: 'desktop', width: 1280, height: 900 },
]

function retryableProjection() {
  return {
    revisions: [],
    generation_run: {
      ID: 'run-1', RunType: 'CHAPTER_GENERATION', StoryID: 'story-1', ChapterID: 'chapter-1',
      Status: 'FAILED', WaitingReason: '', WorkflowVersion: 'v1', Priority: 0,
      BaseCanonVersionID: '', ContextSnapshotID: '', RequestedBy: 'admin-1', IdempotencyKey: 'chapter-1-generation',
    },
    generation_job: {
      ID: 'job-1', RunID: 'run-1', JobType: 'WRITER', Status: 'FAILED', AttemptCount: 1, MaxAttempts: 3,
      LastErrorClass: 'TRANSIENT', LastErrorCode: 'PROVIDER_TIMEOUT', Observation: 'retryable', Retryable: true, AttemptsExhausted: false,
    },
  }
}

async function mockAPI(page, { exhausted = false } = {}) {
  let retryCalls = 0
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname.replace('/api/v1', '')
    const method = route.request().method()
    const json = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
    const notFound = () => json({ error: { code: 'NOT_FOUND', message: 'not found' } }, 404)

    if (path === '/auth/me') return json({ id: 'admin-1', email: 'admin@example.test', roles: ['ADMIN'], status: 'ACTIVE', email_verified: true, mfa_enabled: false })
    if (path === '/admin/stories/story-1/chapters') return json({ chapters: [chapter] })
    if (path === '/admin/stories/story-1/workflow-settings') return json({ story_id: 'story-1', batch_generation_size: 1, creative_autonomy: 'ASSISTED', preferred_text_provider: 'mock', preferred_text_model: 'mock', preferred_tts_provider: 'mock', preferred_voice_id: 'voice-1' })
    if (path === '/admin/chapters/chapter-1/content') {
      const projection = retryableProjection()
      if (exhausted) projection.generation_job = { ...projection.generation_job, AttemptCount: 3, MaxAttempts: 3, LastErrorClass: 'RETRY_EXHAUSTED', LastErrorCode: 'MAX_ATTEMPTS_EXHAUSTED', Observation: 'exhausted', Retryable: false, AttemptsExhausted: true }
      return json(projection)
    }
    if (path === '/admin/chapters/chapter-1/reviews') return json({ reviews: [] })
    if (path === '/admin/chapters/chapter-1/narration/latest') return notFound()
    if (path === '/admin/chapters/chapter-1/audio') return notFound()
    if (path === '/admin/chapters/chapter-1/publish-readiness') return json({ ready: false, missing: ['approved_content', 'narration', 'active_audio'] })
    if (path === '/admin/generation-jobs/job-1/retry' && method === 'POST') {
      retryCalls += 1
      return json({ ...retryableProjection().generation_job, Status: 'PENDING', Observation: 'queued', Retryable: false })
    }
    return notFound()
  })
  return () => retryCalls
}

async function expectNoHorizontalOverflow(page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)).toBe(false)
}

async function focusWithTab(page, locator) {
  await page.locator('body').click({ position: { x: 1, y: 1 } })
  for (let step = 0; step < 24; step += 1) {
    await page.keyboard.press('Tab')
    if (await locator.evaluate((element) => document.activeElement === element)) return
  }
  throw new Error('Retry Generation was not reachable with keyboard Tab traversal')
}

function generationPipelineItem(page) {
  return page
    .getByRole('list', { name: 'Chapter production pipeline' })
    .getByRole('listitem')
    .filter({ has: page.getByRole('heading', { name: 'Generation', exact: true }) })
}

for (const viewport of viewports) {
  test(`authoritative generation retry is responsive and keyboard reachable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    const retryCalls = await mockAPI(page)
    await page.goto('/admin/stories/story-1/production')

    const generationItem = generationPipelineItem(page)
    await expect(generationItem.getByText('RETRYABLE · TRANSIENT · attempt 1/3')).toBeVisible()
    const retry = page.getByRole('button', { name: 'Retry Generation' })
    await expect(retry).toBeVisible()
    await focusWithTab(page, retry)
    await expect(retry).toBeFocused()
    await expectNoHorizontalOverflow(page)

    await retry.click()
    await expect(generationItem.getByText('QUEUED · attempt 1/3')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Retry Generation' })).toHaveCount(0)
    expect(retryCalls()).toBe(1)
  })
}

test('exhausted generation fails closed and never exposes Retry Generation', async ({ page }) => {
  await mockAPI(page, { exhausted: true })
  await page.goto('/admin/stories/story-1/production')
  const generationItem = generationPipelineItem(page)
  await expect(generationItem.getByText('EXHAUSTED · MAX_ATTEMPTS_EXHAUSTED · attempt 3/3')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Retry Generation' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Refresh Generation' })).toBeVisible()
})
