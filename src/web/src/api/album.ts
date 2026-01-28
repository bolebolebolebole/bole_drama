import request from '../utils/request'

// Character Album API
export const characterAlbumAPI = {
  // Delete character image from album
  deleteImage(characterId: string, data: { url: string; scope: 'global' | 'episode'; episode_id?: number }) {
    return request.delete(`/characters/${characterId}/images/delete`, { data })
  },

  // Set primary image for character
  setPrimaryImage(characterId: string, data: { url: string; scope: 'global' | 'episode'; episode_id?: number }) {
    return request.put(`/characters/${characterId}/images/primary`, data)
  }
}

// Scene Album API
export const sceneAlbumAPI = {
  // Upload scene image (multipart form data)
  uploadImage(sceneId: string, formData: FormData) {
    return request.post<{ url: string; filename: string; size: number }>(
      `/scenes/${sceneId}/upload-image`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      }
    )
  },

  // Delete scene image from album
  deleteImage(sceneId: string, data: { url: string }) {
    return request.delete(`/scenes/${sceneId}/images/delete`, { data })
  },

  // Set primary image for scene
  setPrimaryImage(sceneId: string, data: { url: string }) {
    return request.put(`/scenes/${sceneId}/images/primary`, data)
  }
}
