import request from '../utils/request'

export interface EpisodeReferenceAlbumsResponse {
  episode_id: number
  characters: Record<
    string,
    {
      primary_image_url?: string
      global_images: string[]
      episode_primary_image_url?: string
      episode_images: string[]
    }
  >
  scenes: Record<
    string,
    {
      primary_image_url?: string
      images: string[]
    }
  >
}

export const referenceAlbumsAPI = {
  getEpisodeReferenceAlbums(episodeId: string | number) {
    return request.get<EpisodeReferenceAlbumsResponse>(`/episodes/${episodeId}/reference-albums`)
  }
}
