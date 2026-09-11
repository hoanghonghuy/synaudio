import type { CanonApiBoundary } from './canonApiBoundary.mjs'
import type { CanonWorkspaceState } from './canonWorkspacePresentation.mjs'

export interface CanonApprovedRevision {
  ID?: string | null
}

export interface CanonWorkspaceSelection {
  storyID?: string | null
  chapterID?: string | null
  approvedRevision?: CanonApprovedRevision | null
}

export interface CanonWorkspaceController {
  snapshot(): CanonWorkspaceState
  load(selection?: CanonWorkspaceSelection | null): Promise<CanonWorkspaceState>
  commit(): Promise<CanonWorkspaceState>
  reset(): CanonWorkspaceState
}

export function createCanonWorkspaceController(api: CanonApiBoundary): CanonWorkspaceController
