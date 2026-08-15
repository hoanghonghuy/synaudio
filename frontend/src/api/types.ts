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
