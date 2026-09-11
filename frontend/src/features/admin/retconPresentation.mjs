const ACTIONS = Object.freeze({
  DRAFT: ['analyze', 'cancel'],
  ANALYZING: ['approve', 'cancel'],
  APPROVED: ['ready', 'cancel'],
  READY_TO_APPLY: ['apply', 'cancel'],
  APPLIED: [],
  CANCELLED: [],
})

export function retconActions(status) {
  return ACTIONS[status] ?? []
}

export function retconStatusLabel(status) {
  const labels = {
    DRAFT: 'Bản nháp',
    ANALYZING: 'Đang phân tích',
    APPROVED: 'Đã phê duyệt',
    READY_TO_APPLY: 'Sẵn sàng áp dụng',
    APPLIED: 'Đã áp dụng',
    CANCELLED: 'Đã hủy',
  }
  return labels[status] ?? status
}

export function retconActionLabel(action) {
  const labels = {
    analyze: 'Phân tích',
    approve: 'Phê duyệt',
    ready: 'Chuẩn bị áp dụng',
    apply: 'Áp dụng thay đổi',
    cancel: 'Hủy yêu cầu',
  }
  return labels[action] ?? action
}

export function retconActionIsDestructive(action) {
  return action === 'apply' || action === 'cancel'
}
