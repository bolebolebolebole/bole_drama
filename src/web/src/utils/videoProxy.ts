const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

/**
 * 获取视频URL - 优先使用本地路径
 * @param originalUrl 原始视频URL或本地路径
 * @returns 可用的视频URL
 */
export function getVideoProxyUrl(originalUrl: string): string {
  if (!originalUrl) {
    return ''
  }

  // 如果已经是代理URL，直接返回
  if (originalUrl.startsWith('/api/v1/videos/proxy')) {
    return originalUrl
  }

  // 优先使用本地路径（/static/开头）
  if (originalUrl.startsWith('/static/')) {
    return originalUrl
  }

  // 如果是外部URL，使用代理
  if (originalUrl.startsWith('http://') || originalUrl.startsWith('https://')) {
    const encodedUrl = encodeURIComponent(originalUrl)
    return `${API_BASE_URL}/videos/proxy?url=${encodedUrl}`
  }

  return originalUrl
}

export function getVideoProxyRangeUrl(originalUrl: string): string {
  if (!originalUrl) {
    return ''
  }

  if (originalUrl.startsWith('http://') || originalUrl.startsWith('https://')) {
    const encodedUrl = encodeURIComponent(originalUrl)
    return `${API_BASE_URL}/videos/proxy/range?url=${encodedUrl}`
  }

  return getVideoProxyUrl(originalUrl)
}

export function isExternalVideoUrl(url: string): boolean {
  return url?.startsWith('http://') || url?.startsWith('https://')
}

export function shouldUseProxy(url: string): boolean {
  if (!url) return false
  return isExternalVideoUrl(url)
}
