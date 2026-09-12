import { authenticatedRequest } from './client'
import type { CreativeDecision } from './types'

export function selectCreativeDecision(decisionID: string): Promise<CreativeDecision> {
  return authenticatedRequest<CreativeDecision>(`/admin/creative-decisions/${decisionID}/select`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export function rejectCreativeDecision(decisionID: string, scope: string): Promise<CreativeDecision> {
  return authenticatedRequest<CreativeDecision>(`/admin/creative-decisions/${decisionID}/reject`, {
    method: 'POST',
    body: JSON.stringify({ scope }),
  })
}

export function postponeCreativeDecision(decisionID: string, reason: string): Promise<CreativeDecision> {
  return authenticatedRequest<CreativeDecision>(`/admin/creative-decisions/${decisionID}/postpone`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}
