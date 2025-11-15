import Head from 'next/head'
import { GetStaticProps } from 'next'
import { fetchLetterboxdActivity, fetchGoodreadsActivity, fetchGithubActivity } from '@/lib/api'

interface Activity {
  letterboxd: Array<{ title: string; date: string; link: string }>;
  goodreads: Array<{ title: string; author?: string; link: string }>;
  github: Array<{ type: string; repo?: string; description: string; date: string; link: string }>;
}

interface HomeProps {
  activity: Activity;
}

export default function Home({ activity }: HomeProps) {
  return (
    <>
      <Head>
        <title>Garden</title>
        <meta name="description" content="My digital garden" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      <main>
        <header>
          <h1>Garden</h1>
        </header>

        <section className="about">
          <h2>About</h2>
          <p>
            This is my corner of the internet. A simple space where I share what I&apos;m reading, 
            watching, and building. Nothing fancy, just a place to exist online.
          </p>
          <p>
            I believe in keeping things simple and focused. This page reflects that philosophy—
            minimal design, maximum clarity.
          </p>
        </section>

        <section className="activity">
          <h2>What I&apos;m Doing</h2>

          <div className="activity-section">
            <h3>Watching</h3>
            {activity.letterboxd.length > 0 ? (
              <ul>
                {activity.letterboxd.map((item, index) => (
                  <li key={index}>
                    <a href={item.link} target="_blank" rel="noopener noreferrer">
                      {item.title}
                    </a>
                    {item.date && <span className="date"> — {item.date}</span>}
                  </li>
                ))}
              </ul>
            ) : (
              <p>Nothing recent</p>
            )}
            <p className="profile-link">
              <a href={`https://letterboxd.com/${process.env.NEXT_PUBLIC_LETTERBOXD_USERNAME || 'username'}/`} 
                 target="_blank" 
                 rel="noopener noreferrer">
                View all on Letterboxd →
              </a>
            </p>
          </div>

          <div className="activity-section">
            <h3>Reading</h3>
            {activity.goodreads.length > 0 ? (
              <ul>
                {activity.goodreads.map((item, index) => (
                  <li key={index}>
                    <a href={item.link} target="_blank" rel="noopener noreferrer">
                      {item.title}
                    </a>
                    {item.author && <span className="author"> by {item.author}</span>}
                  </li>
                ))}
              </ul>
            ) : (
              <p>Nothing recent</p>
            )}
            <p className="profile-link">
              <a href={`https://www.goodreads.com/user/show/${process.env.NEXT_PUBLIC_GOODREADS_USER_ID || 'userid'}`} 
                 target="_blank" 
                 rel="noopener noreferrer">
                View all on Goodreads →
              </a>
            </p>
          </div>

          <div className="activity-section">
            <h3>Building</h3>
            {activity.github.length > 0 ? (
              <ul>
                {activity.github.map((item, index) => (
                  <li key={index}>
                    <a href={item.link} target="_blank" rel="noopener noreferrer">
                      {item.repo && <span className="repo">{item.repo}</span>}
                      {item.description && <span> — {item.description}</span>}
                    </a>
                    {item.date && <span className="date"> — {item.date}</span>}
                  </li>
                ))}
              </ul>
            ) : (
              <p>Nothing recent</p>
            )}
            <p className="profile-link">
              <a href={`https://github.com/${process.env.NEXT_PUBLIC_GITHUB_USERNAME || 'username'}`} 
                 target="_blank" 
                 rel="noopener noreferrer">
                View all on GitHub →
              </a>
            </p>
          </div>
        </section>

        <footer>
          <p>Last updated: {new Date().getFullYear()}</p>
        </footer>
      </main>
    </>
  )
}

export const getStaticProps: GetStaticProps = async () => {
  const [letterboxd, goodreads, github] = await Promise.all([
    fetchLetterboxdActivity().catch(() => []),
    fetchGoodreadsActivity().catch(() => []),
    fetchGithubActivity().catch(() => []),
  ])

  return {
    props: {
      activity: {
        letterboxd,
        goodreads,
        github,
      },
    },
  }
}

