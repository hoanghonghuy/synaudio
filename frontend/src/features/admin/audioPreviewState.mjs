export function createAudioPreviewState() {
  let requestVersion = 0
  let state = { status: 'idle', chapterID: '', assetID: '', url: '', error: '' }

  function snapshot() {
    return { ...state }
  }

  function reset() {
    requestVersion += 1
    state = { status: 'idle', chapterID: '', assetID: '', url: '', error: '' }
    return snapshot()
  }

  function begin(chapterID, assetID) {
    requestVersion += 1
    const version = requestVersion
    state = { status: 'loading', chapterID, assetID, url: '', error: '' }

    return {
      version,
      mayCommit() {
        return version === requestVersion && state.chapterID === chapterID && state.assetID === assetID
      },
      succeed(url) {
        if (!this.mayCommit()) return snapshot()
        state = { status: 'ready', chapterID, assetID, url, error: '' }
        return snapshot()
      },
      fail(message) {
        if (!this.mayCommit()) return snapshot()
        state = { status: 'error', chapterID, assetID, url: '', error: message }
        return snapshot()
      },
    }
  }

  return { begin, reset, snapshot }
}

export function previewAssetForChapter({ activeChapterID, activeAudio, latestReadyAudio }) {
  const candidate = latestReadyAudio ?? activeAudio ?? null
  if (!candidate || candidate.ChapterID !== activeChapterID || candidate.Status !== 'READY') return null
  return candidate
}
