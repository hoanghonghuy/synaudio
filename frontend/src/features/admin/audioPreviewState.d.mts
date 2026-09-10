import type { AudioAsset } from '../../api/types'

export type AudioPreviewSnapshot = {
  status: 'idle' | 'loading' | 'ready' | 'error'
  chapterID: string
  assetID: string
  url: string
  error: string
}

export type AudioPreviewRequest = {
  version: number
  mayCommit(): boolean
  succeed(url: string): AudioPreviewSnapshot
  fail(message: string): AudioPreviewSnapshot
}

export type AudioPreviewState = {
  begin(chapterID: string, assetID: string): AudioPreviewRequest
  reset(): AudioPreviewSnapshot
  snapshot(): AudioPreviewSnapshot
}

export function createAudioPreviewState(): AudioPreviewState

export function previewAssetForChapter(input: {
  activeChapterID: string
  activeAudio: AudioAsset | null
  latestReadyAudio: AudioAsset | null
}): AudioAsset | null
