export interface Genre {
  id: string
  slug: string
  name: string
}

export type StoryStatus = 'DRAFT' | 'ACTIVE' | 'COMPLETED' | 'ARCHIVED'
export type StoryVisibility = 'PRIVATE' | 'PUBLIC'

export interface Story {
  id: string
  slug: string
  title: string
  description: string
  status: StoryStatus
  visibility: StoryVisibility
}

export interface StoryListResponse {
  stories: Story[]
}

export interface GenreListResponse {
  genres: Genre[]
}

export interface GenerationPolicyInput {
  minimum_audio_duration_sec: number
  target_audio_duration_sec: number
  content_origin: string
  language: string
  narration_language: string
}

export interface CreateStoryInput {
  title: string
  description: string
  created_by: string
  policy: GenerationPolicyInput
}

export interface ApiError {
  error: {
    code: string
    message: string
  }
}

export interface Chapter {
  ID: string
  StoryID: string
  ChapterNumber: number
  Title: string
  Status: string
  ArcID: string
}

export interface ChapterListResponse {
  chapters: Chapter[]
}

export interface ChapterContent {
  chapter_id: string
  revision_id: string
  content_text: string
}

export interface AudioAsset {
  ID: string
  ChapterID: string
  VersionNo: number
  SourceNarrationRevisionID: string
  Status: string
  StorageKey: string
  MimeType: string
  SizeBytes: number
  DurationMs: number
  BitrateKbps: number
  Checksum: string
  IsActive: boolean
}

export interface AudioURLResponse {
  url: string
}

export interface Favorite {
  UserID: string
  StoryID: string
}

export interface FavoriteListResponse {
  favorites: Favorite[]
}

export interface ListeningProgress {
  UserID: string
  ChapterID: string
  PositionMs: number
  CompletedAt: string
  LastAudioAssetID: string
  LastPlaybackSessionID: string
  Version: number
  RelistenStatus?: 'NO_RELISTEN_NEEDED' | 'RELISTEN_RECOMMENDED' | 'RELISTEN_REQUIRED'
}

export interface CreativeDecision {
  ID: string
  StoryID: string
  ChapterID: string
  ArcID: string
  Origin: string
  DecisionType: string
  Severity: string
  Status: string
  BlockingLevel: string
  Question: string
  ContextSummary: string
  SelectedBy: string
}

export interface CreativeDecisionListResponse {
  decisions: CreativeDecision[]
}

export interface AttentionItem {
  ID: string
  StoryID: string
  ChapterID: string
  Priority: string
  Kind: string
  Title: string
  Detail: string
  Action: string
  Resolved: boolean
}

export interface AttentionListResponse {
  items: AttentionItem[]
}

export interface ArcCompletionResult {
  ArcID: string
  Complete: boolean
  TotalChapters: number
  CompletedChapters: number
  PendingChapters: string[]
}

export interface ThreadInactivity {
  ThreadID: string
  Title: string
  Importance: string
  EventCount: number
}

export interface ThreadInactivityResponse {
  inactive_threads: ThreadInactivity[]
}

export interface UsageRecord {
  ID: string
  JobID: string
  AttemptNo: number
  Provider: string
  Model: string
  Status: string
  Usage: Record<string, unknown>
  LatencyMs: number
}

export interface UsageListResponse {
  usage: UsageRecord[]
}
