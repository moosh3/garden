import { parseStringPromise } from 'xml2js'

interface LetterboxdItem {
  title: string;
  date: string;
  link: string;
}

interface GoodreadsItem {
  title: string;
  author?: string;
  link: string;
}

interface GithubItem {
  type: string;
  repo?: string;
  description: string;
  date: string;
  link: string;
}

/**
 * Fetch recent activity from Letterboxd RSS feed
 */
export async function fetchLetterboxdActivity(): Promise<LetterboxdItem[]> {
  const username = process.env.NEXT_PUBLIC_LETTERBOXD_USERNAME
  
  if (!username) {
    console.warn('NEXT_PUBLIC_LETTERBOXD_USERNAME not set')
    return []
  }

  try {
    const response = await fetch(`https://letterboxd.com/${username}/rss/`)
    
    if (!response.ok) {
      throw new Error(`Failed to fetch Letterboxd feed: ${response.status}`)
    }

    const xml = await response.text()
    const result = await parseStringPromise(xml)
    
    const items = result.rss?.channel?.[0]?.item || []
    
    return items.slice(0, 5).map((item: any) => ({
      title: item.title?.[0]?.replace(/^.*?\s-\s/, '') || 'Untitled',
      date: item.pubDate?.[0] ? new Date(item.pubDate[0]).toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric',
        year: 'numeric'
      }) : '',
      link: item.link?.[0] || '#',
    }))
  } catch (error) {
    console.error('Error fetching Letterboxd activity:', error)
    return []
  }
}

/**
 * Fetch recent activity from Goodreads RSS feed
 */
export async function fetchGoodreadsActivity(): Promise<GoodreadsItem[]> {
  const userId = process.env.NEXT_PUBLIC_GOODREADS_USER_ID
  
  if (!userId) {
    console.warn('NEXT_PUBLIC_GOODREADS_USER_ID not set')
    return []
  }

  try {
    const response = await fetch(
      `https://www.goodreads.com/review/list_rss/${userId}?shelf=read`
    )
    
    if (!response.ok) {
      throw new Error(`Failed to fetch Goodreads feed: ${response.status}`)
    }

    const xml = await response.text()
    const result = await parseStringPromise(xml)
    
    const items = result.rss?.channel?.[0]?.item || []
    
    return items.slice(0, 5).map((item: any) => {
      const title = item.title?.[0] || 'Untitled'
      const description = item.description?.[0] || ''
      
      // Try to extract author from description
      const authorMatch = description.match(/author:\s*([^<]+)/)
      const author = authorMatch ? authorMatch[1].trim() : undefined
      
      return {
        title,
        author,
        link: item.link?.[0] || '#',
      }
    })
  } catch (error) {
    console.error('Error fetching Goodreads activity:', error)
    return []
  }
}

/**
 * Fetch recent activity from GitHub API
 */
export async function fetchGithubActivity(): Promise<GithubItem[]> {
  const username = process.env.NEXT_PUBLIC_GITHUB_USERNAME
  const token = process.env.GITHUB_TOKEN // Optional, for higher rate limits
  
  if (!username) {
    console.warn('NEXT_PUBLIC_GITHUB_USERNAME not set')
    return []
  }

  try {
    const headers: HeadersInit = {
      'Accept': 'application/vnd.github.v3+json',
    }
    
    if (token) {
      headers['Authorization'] = `token ${token}`
    }

    const response = await fetch(
      `https://api.github.com/users/${username}/events/public`,
      { 
        headers
      }
    )
    
    if (!response.ok) {
      throw new Error(`Failed to fetch GitHub activity: ${response.status}`)
    }

    const events = await response.json()
    
    // Filter and format the most interesting events
    const relevantEvents = events
      .filter((event: any) => 
        ['PushEvent', 'CreateEvent', 'PullRequestEvent', 'ReleaseEvent'].includes(event.type)
      )
      .slice(0, 5)
      .map((event: any) => {
        let description = ''
        
        switch (event.type) {
          case 'PushEvent':
            const commitCount = event.payload?.commits?.length || 0
            description = `Pushed ${commitCount} commit${commitCount !== 1 ? 's' : ''}`
            break
          case 'CreateEvent':
            description = `Created ${event.payload?.ref_type || 'repository'}`
            break
          case 'PullRequestEvent':
            description = `${event.payload?.action} pull request`
            break
          case 'ReleaseEvent':
            description = `Released ${event.payload?.release?.tag_name || 'new version'}`
            break
          default:
            description = event.type.replace('Event', '')
        }
        
        return {
          type: event.type,
          repo: event.repo?.name,
          description,
          date: new Date(event.created_at).toLocaleDateString('en-US', { 
            month: 'short', 
            day: 'numeric'
          }),
          link: `https://github.com/${event.repo?.name}`,
        }
      })
    
    return relevantEvents
  } catch (error) {
    console.error('Error fetching GitHub activity:', error)
    return []
  }
}

