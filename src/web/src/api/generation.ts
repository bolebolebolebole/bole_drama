import type {
  GenerateCharactersRequest,
  ParseScriptRequest,
  ParseScriptResult,
  ParsedCharacter,
  ParsedEpisode,
  ParsedScene
} from '../types/generation'
import type { Storyboard } from '../types/drama'
import request from '../utils/request'

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

const parseScriptLocally = (data: ParseScriptRequest): ParseScriptResult => {
  // Minimal deterministic parser (no AI) to keep the feature functional.
  // - Episodes: split by "第X集" markers if present; otherwise single episode.
  // - Scenes: split by blank lines.
  const raw = data.script_content || ''
  const normalized = raw.replace(/\r\n/g, '\n').trim()

  const episodes: ParsedEpisode[] = []
  const characters: ParsedCharacter[] = []

  const episodeMatches = Array.from(normalized.matchAll(/(?:^|\n)\s*第\s*(\d+)\s*集\s*[:：-]?\s*(.*)\s*\n/g))
  if (episodeMatches.length === 0 || !data.auto_split) {
    episodes.push({
      episode_number: 1,
      title: '第一集',
      description: '',
      script_content: normalized,
      duration: 0,
      scenes: buildScenes(normalized)
    })
  } else {
    const indexes = episodeMatches.map((m) => ({
      start: m.index ?? 0,
      num: Number(m[1]),
      title: (m[2] || '').trim() || `第${m[1]}集`
    }))
    for (let i = 0; i < indexes.length; i++) {
      const start = indexes[i].start
      const end = i + 1 < indexes.length ? indexes[i + 1].start : normalized.length
      const content = normalized.slice(start, end).trim()
      episodes.push({
        episode_number: indexes[i].num,
        title: indexes[i].title,
        description: '',
        script_content: content,
        duration: 0,
        scenes: buildScenes(content)
      })
    }
  }

  // very naive character extraction: match "姓名：" or "姓名:" at line start
  const names = new Set<string>()
  for (const m of normalized.matchAll(/(?:^|\n)\s*([\u4e00-\u9fa5A-Za-z0-9_]{1,20})\s*[：:]\s*/g)) {
    const name = (m[1] || '').trim()
    if (name && name.length <= 20) names.add(name)
  }
  for (const name of names) {
    characters.push({
      name,
      role: 'unknown',
      description: '',
      personality: ''
    })
  }

  return {
    episodes,
    characters,
    summary: ''
  }
}

function buildScenes(content: string): ParsedScene[] {
  const blocks = content
    .split(/\n\s*\n+/)
    .map((s) => s.trim())
    .filter(Boolean)

  return blocks.map((block, idx) => {
    // Heuristics: use first line as title if short
    const lines = block.split('\n').map((l) => l.trim()).filter(Boolean)
    const first = lines[0] || ''
    const title = first.length <= 40 ? first : `场景 ${idx + 1}`
    const dialogue = lines.slice(1).join('\n')
    return {
      storyboard_number: idx + 1,
      title,
      location: '',
      time: '',
      characters: '',
      dialogue,
      description: block
    }
  })
}

export const generationAPI = {
  generateCharacters(data: GenerateCharactersRequest) {
    return request.post<{ task_id: string; status: string; message: string }>('/generation/characters', data)
  },

  parseScript(data: ParseScriptRequest) {
    // Backend route not present in this package; keep UI usable via local parsing.
    return Promise.resolve(parseScriptLocally(data))
  },

  async generateShots(data: { episode_id: string; script_content?: string; model?: string }) {
    // Align with backend: POST /episodes/:episode_id/storyboards triggers async generation.
    const task = await request.post<{ task_id: string; status: string; message: string }>(
      `/episodes/${data.episode_id}/storyboards`,
      { model: data.model }
    )

    // Poll task status until completion.
    const deadline = Date.now() + 10 * 60 * 1000
    while (Date.now() < deadline) {
      const status = await request.get<{ status: string; error?: string; message?: string }>(`/tasks/${task.task_id}`)
      if (status.status === 'completed') break
      if (status.status === 'failed') {
        throw new Error(status.error || status.message || '生成失败')
      }
      await sleep(1500)
    }

    // Fetch latest storyboards for episode.
    const result = await request.get<{ storyboards: Storyboard[] }>(`/episodes/${data.episode_id}/storyboards`)
    return { shots: result.storyboards || [] }
  },

  generateStoryboard(episodeId: string, model?: string) {
    return request.post<{ task_id: string; status: string; message: string }>(`/episodes/${episodeId}/storyboards`, { model })
  },

  getTaskStatus(taskId: string) {
    return request.get<{
      id: string
      type: string
      status: string
      progress: number
      message?: string
      error?: string
      result?: string
      created_at: string
      updated_at: string
      completed_at?: string
    }>(`/tasks/${taskId}`)
  }
  
}
