import assert from 'node:assert/strict'
import test from 'node:test'
import {
  formatChapterStatus,
  formatChapterTitle,
  formatCreatorLabel,
  formatRevisionStatus,
  formatSourceType,
} from '../src/features/admin/reviewPresentation.mjs'

test('formatChapterTitle strips redundant chapter number prefixes', () => {
  assert.equal(
    formatChapterTitle('Chương 1: Tiếng Gọi Dưới Đáy Sâu', 1),
    'Tiếng Gọi Dưới Đáy Sâu',
  )
  assert.equal(
    formatChapterTitle('Chương 1 - Tiếng Gọi Dưới Đáy Sâu', 1),
    'Tiếng Gọi Dưới Đáy Sâu',
  )
  assert.equal(
    formatChapterTitle('Chương 2. Bí Ẩn Hồ Tây', 2),
    'Bí Ẩn Hồ Tây',
  )
  assert.equal(
    formatChapterTitle('Tiếng Gọi Dưới Đáy Sâu', 1),
    'Tiếng Gọi Dưới Đáy Sâu',
  )
  assert.equal(
    formatChapterTitle('Chương 1', 1),
    'Chương 1',
  )
})

test('formatCreatorLabel masks raw internal UUIDs and formats current user', () => {
  const uuid = 'bef28055-17f4-49dd-b114-dd1d311e4b1f'
  assert.equal(formatCreatorLabel(uuid, 'other-id'), 'Tác giả')
  assert.equal(formatCreatorLabel(uuid, uuid), 'Tôi')
  assert.equal(formatCreatorLabel('', 'other-id'), 'Hệ thống')
  assert.equal(formatCreatorLabel(null, 'other-id'), 'Hệ thống')
  assert.equal(formatCreatorLabel('admin@synaudio.app', 'other-id'), 'admin@synaudio.app')
})

test('formatSourceType provides clear localized labels', () => {
  assert.equal(formatSourceType('AI_GENERATED'), 'AI sinh nội dung')
  assert.equal(formatSourceType('MANUAL_EDIT'), 'Biên tập viên chỉnh sửa')
  assert.equal(formatSourceType('HUMAN_WRITTEN'), 'Tác giả tự viết')
})

test('formatChapterStatus and formatRevisionStatus provide Vietnamese labels', () => {
  assert.equal(formatChapterStatus('PUBLISHED'), 'Đã phát hành')
  assert.equal(formatChapterStatus('READY'), 'Sẵn sàng phát')
  assert.equal(formatChapterStatus('IN_PRODUCTION'), 'Đang sản xuất')
  assert.equal(formatRevisionStatus('APPROVED'), 'Đã duyệt')
  assert.equal(formatRevisionStatus('CANDIDATE'), 'Chờ duyệt')
  assert.equal(formatRevisionStatus('REJECTED'), 'Đã từ chối')
})

test('ContentReviewPage uses review presentation formatting to prevent title and badge duplication', async () => {
  const { readFile } = await import('node:fs/promises')
  const reviewPage = await readFile(new URL('../src/features/admin/ContentReviewPage.vue', import.meta.url), 'utf8')
  assert.match(reviewPage, /formatChapterTitle/)
  assert.match(reviewPage, /formatCreatorLabel/)
  assert.match(reviewPage, /formatSourceType/)
  assert.match(reviewPage, /formatChapterStatus/)
})

