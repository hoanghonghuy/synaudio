import { authenticatedRequest } from './client'

export type RetconStatus = 'DRAFT' | 'ANALYZING' | 'APPROVED' | 'READY_TO_APPLY' | 'APPLIED' | 'CANCELLED'

export type RetconRequest = {
  ID: string
  StoryID: string
  TargetChapterID: string
  Status: RetconStatus
  ImpactScope: string
  ProposedChange: string
  Reason: string
  RequestedBy: string
  ApprovedBy: string
  AppliedBy: string
}

export function listRetcons(storyID: string): Promise<{ retcons: RetconRequest[] }> {
  return authenticatedRequest<{ retcons: RetconRequest[] }>(`/admin/retcons?story_id=${encodeURIComponent(storyID)}`)
}

export function getRetcon(id: string): Promise<RetconRequest> {
  return authenticatedRequest<RetconRequest>(`/admin/retcons/${encodeURIComponent(id)}`)
}

function mutateRetcon(id: string, action: 'analyze' | 'approve' | 'ready' | 'apply' | 'cancel'): Promise<RetconRequest> {
  const needsBody = action === 'approve' || action === 'apply'
  return authenticatedRequest<RetconRequest>(`/admin/retcons/${encodeURIComponent(id)}/${action}`, {
    method: 'POST',
    ...(needsBody ? { body: JSON.stringify({}) } : {}),
  })
}

export function analyzeRetcon(id: string): Promise<RetconRequest> { return mutateRetcon(id, 'analyze') }
export function approveRetcon(id: string): Promise<RetconRequest> { return mutateRetcon(id, 'approve') }
export function markRetconReady(id: string): Promise<RetconRequest> { return mutateRetcon(id, 'ready') }
export function applyRetcon(id: string): Promise<RetconRequest> { return mutateRetcon(id, 'apply') }
export function cancelRetcon(id: string): Promise<RetconRequest> { return mutateRetcon(id, 'cancel') }
