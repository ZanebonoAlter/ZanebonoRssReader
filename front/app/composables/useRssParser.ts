import type { RssFeed, Article, FeedResponse } from '~/types'

export function useRssParser() {
  const config = useRuntimeConfig()
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Parse RSS feed from URL
   * Uses a CORS proxy to fetch RSS feeds
   */
  async function fetchFeed(url: string): Promise<FeedResponse | null> {
    loading.value = true
    error.value = null

    try {
      // Using RSS2JSON API as a CORS proxy
      const apiUrl = `https://api.rss2json.com/v1/api.json?rss_url=${encodeURIComponent(url)}`

      const response = await $fetch<FeedResponse>(apiUrl)

      if (response.status !== 'ok') {
        throw new Error('Failed to parse RSS feed')
      }

      return response
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch feed'
      console.error('Error fetching RSS feed:', e)
      return null
    } finally {
      loading.value = false
    }
  }

  /**
   * Convert RSS2JSON response to our Article format
   */
  function convertToArticles(feedId: string, feedData: FeedResponse): Article[] {
    return feedData.items.map((item, index) => ({
      id: `${feedId}-${index}`,
      feedId,
      title: item.title,
      description: cleanDescription(item.description || ''),
      content: item.content || item.description || '',
      link: item.link,
      pubDate: item.pubDate,
      author: item.author,
      category: feedId, // Will be updated by the caller
      read: false,
      favorite: false,
      imageUrl: item.thumbnail || item.enclosure?.link || extractFirstImage(item.content || '')
    }))
  }

  /**
   * Create a new RssFeed object from RSS2JSON response
   */
  function createFeedFromResponse(
    url: string,
    categoryId: string,
    feedData: FeedResponse
  ): Omit<RssFeed, 'id' | 'articleCount'> {
    return {
      title: feedData.feed.title,
      description: feedData.feed.description || '',
      url,
      category: categoryId,
      icon: 'mdi:rss',
      color: getCategoryColor(categoryId),
      lastUpdated: new Date().toISOString()
    }
  }

  return {
    loading: readonly(loading),
    error: readonly(error),
    fetchFeed,
    convertToArticles,
    createFeedFromResponse
  }
}

/**
 * Helper function to clean HTML from descriptions
 */
function cleanDescription(html: string): string {
  if (!html) return ''

  // Remove HTML tags
  const text = html.replace(/<[^>]*>/g, '')

  // Decode HTML entities
  const textarea = document.createElement('textarea')
  textarea.innerHTML = text
  return textarea.value.substring(0, 200) // Limit to 200 characters
}

/**
 * Helper function to extract first image from HTML content
 */
function extractFirstImage(html: string): string | undefined {
  if (!html) return undefined

  const imgMatch = html.match(/<img[^>]+src="([^">]+)"/i)
  return imgMatch?.[1] || undefined
}

/**
 * Helper function to get color for category
 */
function getCategoryColor(categoryId: string): string {
  const colors: Record<string, string> = {
    tech: '#3b82f6',
    news: '#ef4444',
    design: '#8b5cf6',
    blog: '#10b981',
    ai: '#f59e0b',
    product: '#ec4899'
  }

  return colors[categoryId] || '#6b7280'
}
