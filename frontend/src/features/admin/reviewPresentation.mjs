/**
 * Presentation helpers for content review, chapter titles, creator identities, and status formatting.
 */

const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

export function formatChapterTitle(rawTitle, chapterNumber) {
  if (!rawTitle || typeof rawTitle !== 'string') {
    return chapterNumber ? `Chương ${chapterNumber}` : ''
  }
  const trimmed = rawTitle.trim()
  const cleaned = trimmed.replace(/^(?:Chương|Hồi|Tập|Chapter)\s*\d+[\s:.-]*/i, '').trim()
  return cleaned || trimmed
}

export function formatCreatorLabel(createdBy, currentUserId) {
  if (!createdBy || typeof createdBy !== 'string' || !createdBy.trim()) {
    return 'Hệ thống'
  }
  const trimmed = createdBy.trim()
  if (currentUserId && trimmed === currentUserId) {
    return 'Tôi'
  }
  if (UUID_REGEX.test(trimmed)) {
    return 'Tác giả'
  }
  return trimmed
}

export function formatSourceType(sourceType) {
  const labels = {
    AI_GENERATED: 'AI sinh nội dung',
    MANUAL_EDIT: 'Biên tập viên chỉnh sửa',
    HUMAN_WRITTEN: 'Tác giả tự viết',
  }
  return labels[sourceType] ?? (sourceType || 'Chưa rõ')
}

export function formatChapterStatus(status) {
  const labels = {
    PUBLISHED: 'Đã phát hành',
    READY: 'Sẵn sàng phát',
    IN_PRODUCTION: 'Đang sản xuất',
    PLANNED: 'Đã lên kế hoạch',
    DRAFT: 'Bản nháp',
  }
  return labels[status] ?? (status || 'Chưa rõ')
}

export function formatRevisionStatus(status) {
  const labels = {
    CANDIDATE: 'Chờ duyệt',
    APPROVED: 'Đã duyệt',
    REJECTED: 'Đã từ chối',
    SUPERSEDED: 'Đã thay thế',
  }
  return labels[status] ?? (status || 'Chưa rõ')
}
