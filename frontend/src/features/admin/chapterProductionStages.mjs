const LABELS = {
  plan: 'Plan',
  generation: 'Generation',
  review: 'Review',
  canon: 'Canon / Memory',
  narration: 'Narration',
  audio: 'Audio',
  publish: 'Publish',
}

function stage(id, state, summary, blocker = '') {
  return { id, label: LABELS[id], state, summary, blocker }
}

export function buildChapterProductionStages(input) {
  const {
    selectionLoading = false,
    hasPlanRevision = false,
    generationJobStatus = '',
    hasApprovedRevision = false,
    canonStatus = '',
    narrationStatus = '',
    audioStatus = '',
    publishStatus = '',
    chapterPublished = false,
  } = input

  if (selectionLoading) {
    return Object.keys(LABELS).map((id) => stage(id, 'loading', 'Đang tải trạng thái authoritative…'))
  }

  const plan = hasPlanRevision
    ? stage('plan', 'ready', 'Plan revision hiện hành đã sẵn sàng.')
    : stage('plan', 'blocked', 'Chưa có plan revision hiện hành.', 'Hoàn thiện Story Planning trước khi generation.')

  const generation = generationJobStatus.startsWith('FAILED') || generationJobStatus.startsWith('BLOCKED')
    ? stage('generation', 'failed', generationJobStatus, 'Kiểm tra lỗi generation và dùng Retry khi backend cho phép.')
    : generationJobStatus.startsWith('RUNNING') || generationJobStatus.startsWith('QUEUED') || generationJobStatus.startsWith('WAITING')
      ? stage('generation', 'processing', generationJobStatus)
      : generationJobStatus
        ? stage('generation', 'ready', generationJobStatus)
        : stage('generation', hasPlanRevision ? 'waiting' : 'blocked', 'Chưa có generation output.', hasPlanRevision ? '' : 'Cần plan revision trước.')

  const review = hasApprovedRevision
    ? stage('review', 'ready', 'Approved content authoritative đã sẵn sàng.')
    : stage('review', 'waiting', 'Chưa có approved content.', 'Mở Content Review và approve revision trước khi commit Canon/Memory.')

  const canon = canonStatus.startsWith('BLOCKED')
    ? stage('canon', 'blocked', canonStatus, canonStatus.replace(/^BLOCKED\s*[—-]?\s*/, ''))
    : canonStatus.startsWith('CURRENT')
      ? stage('canon', 'ready', canonStatus)
      : canonStatus.startsWith('READY TO COMMIT')
        ? stage('canon', 'waiting', canonStatus, 'Commit Canon/Memory cho exact approved revision trước khi tiếp tục production.')
        : canonStatus.startsWith('WAITING')
          ? stage('canon', 'waiting', canonStatus)
          : stage('canon', hasApprovedRevision ? 'waiting' : 'blocked', canonStatus || 'Chưa có Canon/Memory authority.', hasApprovedRevision ? 'Đang chờ Canon/Memory authority cho approved revision.' : 'Cần approved content trước.')

  const narration = narrationStatus.startsWith('BLOCKED')
    ? stage('narration', 'blocked', narrationStatus, narrationStatus.replace(/^BLOCKED\s*[—-]?\s*/, ''))
    : narrationStatus.startsWith('READY')
      ? stage('narration', 'ready', narrationStatus)
      : narrationStatus.startsWith('WAITING')
        ? stage('narration', 'waiting', narrationStatus)
        : stage('narration', narrationStatus ? 'ready' : 'waiting', narrationStatus || 'Chưa có narration authoritative.')

  const audio = audioStatus.startsWith('BLOCKED')
    ? stage('audio', 'blocked', audioStatus, audioStatus.replace(/^BLOCKED\s*[—-]?\s*/, ''))
    : audioStatus.startsWith('ACTIVE')
      ? stage('audio', 'active', audioStatus)
      : audioStatus.startsWith('READY')
        ? stage('audio', 'ready', audioStatus)
        : audioStatus.startsWith('FAILED')
          ? stage('audio', 'failed', audioStatus, 'Retry/synthesize lại chỉ khi backend cho phép; không tạo duplicate asset.')
          : stage('audio', 'waiting', audioStatus || 'Chưa có READY audio durable.')

  const publish = chapterPublished || publishStatus.startsWith('PUBLISHED')
    ? stage('publish', 'published', publishStatus || 'Chapter đã publish.')
    : publishStatus.startsWith('BLOCKED')
      ? stage('publish', 'blocked', publishStatus, publishStatus.replace(/^BLOCKED\s*[—-]?\s*/, ''))
      : publishStatus.startsWith('READY')
        ? stage('publish', 'ready', publishStatus)
        : stage('publish', 'waiting', publishStatus || 'Chưa có publish readiness authoritative.')

  return [plan, generation, review, canon, narration, audio, publish]
}

export function getPrimaryProductionStage(stages) {
  return stages.find((item) => !['ready', 'active', 'published'].includes(item.state))
    ?? stages[stages.length - 1]
    ?? null
}

export function stageStateLabel(state) {
  return {
    loading: 'Đang tải',
    blocked: 'Bị chặn',
    waiting: 'Chờ',
    processing: 'Đang xử lý',
    failed: 'Thất bại',
    ready: 'Sẵn sàng',
    active: 'Đang active',
    published: 'Đã publish',
  }[state] ?? state
}
