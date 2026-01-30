import request from '@/utils/request'

export interface SceneLibraryItem {
  id: string | number
  location: string
  time: string
  prompt: string
  category?: string
  image_url: string
  source_type?: string
}

export interface SceneLibraryListResponse {
  items: SceneLibraryItem[]
  total: number
  page: number
  page_size: number
}

export const sceneLibraryAPI = {
  list(params?: { page?: number; page_size?: number; category?: string; keyword?: string }) {
    return request.get<SceneLibraryListResponse>('/scene-library', { params })
  },

  delete(itemId: string | number) {
    return request.delete(`/scene-library/${itemId}`)
  },

  applyFromLibrary(sceneId: string | number, libraryItemId: string | number) {
    return request.put(`/scenes/${sceneId}/image-from-library`, {
      library_item_id: String(libraryItemId)
    })
  },

  addSceneToLibrary(sceneId: string | number, category?: string) {
    return request.post(`/scenes/${sceneId}/add-to-library`, { category })
  },

  // 场景多图管理
  listImages(sceneId: string | number) {
    return request.get<{ items: Array<{ id: number; scene_id: number; image_url: string; sort_order: number; created_at: string; updated_at: string }> }>(
      `/scenes/${sceneId}/images`
    )
  },
  addImage(sceneId: string | number, imageUrl: string) {
    return request.post(`/scenes/${sceneId}/images`, { image_url: imageUrl })
  },
  reorderImages(sceneId: string | number, imageIds: number[]) {
    return request.put(`/scenes/${sceneId}/images/reorder`, { image_ids: imageIds })
  },
  deleteImage(sceneId: string | number, imageId: number) {
    return request.delete(`/scenes/${sceneId}/images/${imageId}`)
  }
}
