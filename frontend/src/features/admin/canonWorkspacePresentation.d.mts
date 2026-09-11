export interface CanonWorkspaceLatestVersion {
  SequenceNo?: number | string | null
  SourceContentRevisionID?: string | null
}

export interface CanonWorkspaceReadiness {
  state?: string
  label?: string
  canCommit?: boolean
  latestVersion?: CanonWorkspaceLatestVersion | null
}

export interface CanonWorkspaceState {
  readiness?: CanonWorkspaceReadiness | null
  error?: string | null
  loading?: boolean
  committing?: boolean
}

export interface CanonWorkspacePresentation {
  status: string
  badge: string
  summary: string
  detail: string
  canCommit: boolean
  canRetry: boolean
  busy: boolean
}

export function presentCanonWorkspace(state?: CanonWorkspaceState): CanonWorkspacePresentation
