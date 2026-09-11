export type CanonAuthenticatedRequest = <T = unknown>(path: string, init?: RequestInit) => Promise<T>

export interface CanonCommitInput {
  storyID?: string | null
  branchID?: string | null
  chapterID?: string | null
  approvedRevisionID?: string | null
}

export interface CanonApiBoundary {
  getActiveOfficialBranch(storyID: string): Promise<Record<string, unknown>>
  listVersions(branchID: string): Promise<unknown>
  commitApprovedRevision(input?: CanonCommitInput): Promise<Record<string, unknown>>
}

export function createCanonApiBoundary(request: CanonAuthenticatedRequest): CanonApiBoundary
