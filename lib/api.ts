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
  description?: string;
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
    const allowedTypes = ['PushEvent', 'CreateEvent', 'PullRequestEvent', 'ReleaseEvent']
    const uniqueRepos: GithubItem[] = []
    const seenRepos = new Set<string>()

    for (const event of events) {
      if (!allowedTypes.includes(event.type)) {
        continue
      }

      const repoName = event.repo?.name
      if (!repoName || seenRepos.has(repoName)) {
        continue
      }

      seenRepos.add(repoName)
      uniqueRepos.push({
        type: event.type,
        repo: repoName,
        date: new Date(event.created_at).toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric',
        }),
        link: `https://github.com/${repoName}`,
      })

      if (uniqueRepos.length === 3) {
        break
      }
    }
    
    return uniqueRepos
  } catch (error) {
    console.error('Error fetching GitHub activity:', error)
    return []
  }
}

