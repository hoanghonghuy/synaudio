function versionSummary(version) {
  if (!version) return ''
  const sequence = Number(version.SequenceNo ?? 0)
  const source = String(version.SourceContentRevisionID ?? '').trim() || '—'
  return `OFFICIAL v${sequence || '—'} · source ${source}`
}

export function presentCanonWorkspace(state = {}) {
  const readiness = state.readiness ?? {}
  const status = String(readiness.state ?? 'BLOCKED')
  const latestVersion = readiness.latestVersion ?? null
  const authorityError = String(state.error ?? '').trim()

  if (status === 'LOADING') return { status, badge: 'LOADING', summary: readiness.label || 'Đang tải Canon/Memory authority…', detail: 'Đang xác minh ACTIVE OFFICIAL branch và CanonVersion authoritative.', canCommit: false, canRetry: false, busy: true }
  if (status === 'WAITING') return { status, badge: 'WAITING', summary: readiness.label || 'WAITING — cần approved content trước khi commit Canon/Memory', detail: 'Approve content revision hiện hành trước khi tạo Canon/Memory projection.', canCommit: false, canRetry: false, busy: false }
  if (status === 'CURRENT') return { status, badge: 'CURRENT', summary: readiness.label || 'CURRENT — Canon/Memory khớp approved revision hiện hành', detail: versionSummary(latestVersion), canCommit: false, canRetry: false, busy: false }
  if (status === 'READY_TO_COMMIT') return { status, badge: 'READY TO COMMIT', summary: readiness.label || 'READY TO COMMIT — Canon/Memory cần cập nhật', detail: latestVersion ? `Current Canon: ${versionSummary(latestVersion)}` : 'Chưa có OFFICIAL CanonVersion cho approved revision hiện hành.', canCommit: Boolean(readiness.canCommit) && !state.committing, canRetry: false, busy: Boolean(state.committing) }

  return { status: 'BLOCKED', badge: 'BLOCKED', summary: authorityError || readiness.label || 'BLOCKED — không xác minh được Canon/Memory authority', detail: 'Không commit khi ACTIVE OFFICIAL authority chưa xác minh. Retry chỉ tải lại authority, không tạo duplicate mutation.', canCommit: false, canRetry: !state.loading && !state.committing, busy: Boolean(state.loading || state.committing) }
}
