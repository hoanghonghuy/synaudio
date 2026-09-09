import { privilegedAssuranceCode } from './http-error.ts'

export type PrivilegedAssuranceReason = 'MFA_REQUIRED' | 'RECENT_AUTH_REQUIRED'

export type PrivilegedReAuthHandler = (reason: PrivilegedAssuranceReason) => Promise<boolean>

/**
 * Runs an admin mutation/read behind the backend privileged-assurance contract.
 * When the server requires MFA or recent-auth for the current session, the handler
 * must complete `/auth/re-auth` before a single explicit retry is attempted.
 */
export async function withPrivilegedRetry<T>(
  operation: () => Promise<T>,
  onReAuthRequired: PrivilegedReAuthHandler,
): Promise<T> {
  try {
    return await operation()
  } catch (error) {
    const reason = privilegedAssuranceCode(error)
    if (!reason) throw error
    const recovered = await onReAuthRequired(reason)
    if (!recovered) throw error
    return await operation()
  }
}
