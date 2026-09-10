export type ChapterProductionStageState =
  | 'loading'
  | 'blocked'
  | 'waiting'
  | 'processing'
  | 'failed'
  | 'ready'
  | 'active'
  | 'published'

export interface ChapterProductionStage {
  id: 'plan' | 'generation' | 'review' | 'narration' | 'audio' | 'publish'
  label: string
  state: ChapterProductionStageState
  summary: string
  blocker: string
}

export interface ChapterProductionStageInput {
  selectionLoading?: boolean
  hasPlanRevision?: boolean
  generationJobStatus?: string
  hasApprovedRevision?: boolean
  narrationStatus?: string
  audioStatus?: string
  publishStatus?: string
  chapterPublished?: boolean
}

export function buildChapterProductionStages(input: ChapterProductionStageInput): ChapterProductionStage[]
export function getPrimaryProductionStage(stages: ChapterProductionStage[]): ChapterProductionStage | null
export function stageStateLabel(state: ChapterProductionStageState | string): string
