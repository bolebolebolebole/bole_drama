import request from '../utils/request'
import type { ImageGeneration } from '../types/image'

export interface CharacterLibraryItem {
  id: string | number
  name: string
  category?: string
  image_url: string
  description?: string
  tags?: string
  source_type: string
  created_at: string
  updated_at: string
}

export interface CreateLibraryItemRequest {
  name: string
  category?: string
  image_url: string
  description?: string
  tags?: string
  source_type?: string
}

export interface CharacterLibraryQuery {
  page?: number
  page_size?: number
  category?: string
  source_type?: string
  keyword?: string
}

export const characterLibraryAPI = {
  // 获取角色库列表
  list(params?: CharacterLibraryQuery) {
    return request.get<{
      items: CharacterLibraryItem[]
      pagination: {
        page: number
        page_size: number
        total: number
        total_pages: number
      }
    }>('/character-library', { params })
  },

  // 创建角色库项
  create(data: CreateLibraryItemRequest) {
    return request.post<CharacterLibraryItem>('/character-library', data)
  },

  // 获取角色库项详情
  get(id: string) {
    return request.get<CharacterLibraryItem>(`/character-library/${id}`)
  },

  // 删除角色库项
  delete(id: string) {
    return request.delete(`/character-library/${id}`)
  },

  // 上传角色图片
  uploadCharacterImage(characterId: string, imageUrl: string) {
    return request.put(`/characters/${characterId}/image`, { image_url: imageUrl })
  },

  // 角色多图管理
  listImages(characterId: string | number) {
    return request.get<{ items: Array<{ id: number; character_id: number; image_url: string; sort_order: number; created_at: string; updated_at: string }> }>(
      `/characters/${characterId}/images`
    )
  },
  addImage(characterId: string | number, imageUrl: string) {
    return request.post(`/characters/${characterId}/images`, { image_url: imageUrl })
  },
  reorderImages(characterId: string | number, imageIds: number[]) {
    return request.put(`/characters/${characterId}/images/reorder`, { image_ids: imageIds })
  },
  deleteImage(characterId: string | number, imageId: number) {
    return request.delete(`/characters/${characterId}/images/${imageId}`)
  },

  // 从角色库应用形象
  applyFromLibrary(characterId: string | number, libraryItemId: string | number) {
    return request.put(`/characters/${characterId}/image-from-library`, {
      library_item_id: String(libraryItemId)
    })
  },

  // 将角色添加到角色库
  addCharacterToLibrary(characterId: string | number, category?: string) {
    return request.post<CharacterLibraryItem>(`/characters/${characterId}/add-to-library`, {
      category
    })
  },

  // AI生成角色形象
  generateCharacterImage(characterId: string, model?: string) {
    return request.post<{ message: string; image_generation: ImageGeneration }>(`/characters/${characterId}/generate-image`, {
      model
    })
  },

  // 批量生成角色形象
  batchGenerateCharacterImages(characterIds: string[], model?: string) {
    return request.post<{ message: string; count: number }>('/characters/batch-generate-images', {
      character_ids: characterIds,
      model
    })
  },

  // 更新角色信息
  updateCharacter(characterId: number, data: {
    name?: string
    role?: string
    appearance?: string
    personality?: string
    description?: string
  }) {
    return request.put(`/characters/${characterId}`, data)
  },

  // 重新从剧本中提取该角色的提示词/设定
  reextractCharacterPrompt(characterId: string | number, data: { episode_id?: number; script_content?: string; model?: string }) {
    return request.post<{ message: string; character: any }>(`/characters/${characterId}/reextract-prompt`, data)
  },

  // 删除角色
  deleteCharacter(characterId: number) {
    return request.delete(`/characters/${characterId}`)
  }
}
