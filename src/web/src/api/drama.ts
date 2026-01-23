import type {
  CreateDramaRequest,
  Drama,
  DramaListQuery,
  DramaStats,
  UpdateDramaRequest
} from '../types/drama'
import type { ImageGeneration } from '../types/image'
import request from '../utils/request'

export const dramaAPI = {
  list(params?: DramaListQuery) {
    return request.get<{
      items: Drama[]
      pagination: {
        page: number
        page_size: number
        total: number
        total_pages: number
      }
    }>('/dramas', { params })
  },

  create(data: CreateDramaRequest) {
    return request.post<Drama>('/dramas', data)
  },

  get(id: string) {
    return request.get<Drama>(`/dramas/${id}`)
  },

  update(id: string, data: UpdateDramaRequest) {
    return request.put<Drama>(`/dramas/${id}`, data)
  },

  delete(id: string) {
    return request.delete(`/dramas/${id}`)
  },

  getStats() {
    return request.get<DramaStats>('/dramas/stats')
  },

  saveOutline(id: string, data: { title: string; summary: string; genre?: string; tags?: string[] }) {
    return request.put(`/dramas/${id}/outline`, data)
  },

  getCharacters(dramaId: string) {
    return request.get(`/dramas/${dramaId}/characters`)
  },

  saveCharacters(id: string, data: any[], episodeId?: string) {
    return request.put(`/dramas/${id}/characters`, {
      characters: data,
      episode_id: episodeId ? parseInt(episodeId) : undefined
    })
  },

  saveEpisodes(id: string, data: any[]) {
    return request.put(`/dramas/${id}/episodes`, { episodes: data })
  },

  saveProgress(id: string, data: { current_step: string; step_data?: any }) {
    return request.put(`/dramas/${id}/progress`, data)
  },

  generateStoryboard(episodeId: string) {
    return request.post(`/episodes/${episodeId}/storyboards`)
  },

  getBackgrounds(episodeId: string) {
    return request.get(`/images/episode/${episodeId}/backgrounds`)
  },

  extractBackgrounds(episodeId: string, model?: string) {
    return request.post<{ task_id: string; status: string; message: string }>(`/images/episode/${episodeId}/backgrounds/extract`, { model })
  },

  batchGenerateBackgrounds(episodeId: string) {
    return request.post(`/images/episode/${episodeId}/batch`)
  },

  generateSingleBackground(backgroundId: number, dramaId: string, prompt: string, referenceImages?: string[]) {
    return request.post('/images', {
      scene_id: backgroundId,
      drama_id: dramaId,
      prompt: prompt,
      reference_images: referenceImages || []
    })
  },

  getStoryboards(episodeId: string) {
    return request.get(`/episodes/${episodeId}/storyboards`)
  },

  insertStoryboard(episodeId: string, insertAfter: number, duration: number = 5) {
    return request.post(`/episodes/${episodeId}/storyboards/insert`, {
      insert_after: insertAfter,
      duration: duration
    })
  },

  updateStoryboard(storyboardId: string, data: any) {
    return request.put(`/storyboards/${storyboardId}`, data)
  },


  updateScene(sceneId: string, data: {
    location?: string
    time?: string
    prompt?: string
  }) {
    return request.put(`/scenes/${sceneId}`, data)
  },

  generateSceneImage(data: { scene_id: number; prompt?: string; model?: string }) {
    return request.post<{ message: string; image_generation: ImageGeneration }>('/scenes/generate-image', data)
  },

  deleteScene(sceneId: string) {
    return request.delete(`/scenes/${sceneId}`)
  },

  // 完成集数制作（触发视频合成）
  finalizeEpisode(episodeId: string, timelineData?: any) {
    return request.post(`/episodes/${episodeId}/finalize`, timelineData || {})
  }
}
