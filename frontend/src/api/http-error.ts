export type ProgressConflictPayload = {
  UserID: string
  ChapterID: string
  PositionMs: number
  CompletedAt: string
  LastAudioAssetID: string
  LastPlaybackSessionID: string
  Version: number
  RelistenStatus?: 'NO_RELISTEN_NEEDED' | 'RELISTEN_RECOMMENDED' | 'RELISTEN_REQUIRED'
}

export class ApiRequestError extends Error {
  readonly status: number
  readonly code?: string
  readonly progress?: ProgressConflictPayload

  constructor(status: number, message: string, code?: string, progress?: ProgressConflictPayload) {
    super(message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = code
    this.progress = progress
  }
}

export function isExplicitNotFound(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 404
}

export function isProgressVersionConflict(error: unknown): error is ApiRequestError & { progress: ProgressConflictPayload } {
  return (
    error instanceof ApiRequestError &&
    error.status === 409 &&
    error.code === 'PROGRESS_VERSION_CONFLICT' &&
    error.progress != null
  )
}

export function isMfaRequired(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 403 && error.code === 'MFA_REQUIRED'
}

export function isRecentAuthRequired(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 403 && error.code === 'RECENT_AUTH_REQUIRED'
}

export function isPrivilegedAssuranceRequired(error: unknown): boolean {
  return isMfaRequired(error) || isRecentAuthRequired(error)
}

export function isLastActiveAdmin(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 409 && error.code === 'LAST_ACTIVE_ADMIN'
}

export function isPermissionDenied(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 403 && error.code === 'FORBIDDEN'
}

export function privilegedAssuranceCode(error: unknown): 'MFA_REQUIRED' | 'RECENT_AUTH_REQUIRED' | null {
  if (isMfaRequired(error)) return 'MFA_REQUIRED'
  if (isRecentAuthRequired(error)) return 'RECENT_AUTH_REQUIRED'
  return null
}

export function adminSecurityErrorMessage(error: unknown): string {
  if (!(error instanceof ApiRequestError)) {
    return error instanceof Error ? error.message : 'Đã xảy ra lỗi không xác định.'
  }
  switch (error.code) {
    case 'MFA_REQUIRED':
      return 'Phiên hiện tại chưa được xác minh MFA. Xác minh lại bằng mã TOTP hoặc mã khôi phục của phiên này.'
    case 'RECENT_AUTH_REQUIRED':
      return 'Thao tác này yêu cầu xác thực gần đây. Xác minh lại bằng mã TOTP hoặc mã khôi phục của phiên hiện tại.'
    case 'FORBIDDEN':
      return 'Bạn không có quyền thực hiện thao tác này. Quyền được xác định bởi máy chủ.'
    case 'LAST_ACTIVE_ADMIN':
      return 'Không thể thực hiện: hệ thống cần giữ ít nhất một Admin đang hoạt động và có MFA.'
    case 'USER_NOT_FOUND':
      return 'Không tìm thấy người dùng. Dữ liệu có thể đã thay đổi — danh sách sẽ được làm mới.'
    case 'INVALID_STATUS':
      return 'Trạng thái tài khoản không hợp lệ.'
    case 'INVALID_TOKEN':
      return 'Mã xác thực không hợp lệ hoặc đã hết hạn.'
    case 'UNAUTHENTICATED':
      return 'Phiên đăng nhập không còn hiệu lực. Vui lòng đăng nhập lại.'
    default:
      return error.message
  }
}
