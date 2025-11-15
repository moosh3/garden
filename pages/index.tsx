import Head from 'next/head'
import Image from 'next/image'
import { GetStaticProps } from 'next'
import { fetchLetterboxdActivity, fetchGoodreadsActivity, fetchGithubActivity } from '@/lib/api'

interface Activity {
  letterboxd: Array<{ title: string; date: string; link: string }>;
  goodreads: Array<{ title: string; author?: string; link: string }>;
  github: Array<{ type: string; repo?: string; description?: string; date: string; link: string }>;
}

interface HomeProps {
  activity: Activity;
}

export default function Home({ activity }: HomeProps) {
  return (
    <>
      <Head>
        <title>Alec</title>
        <meta name="description" content="My digital garden" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
      </Head>
      <main>
        <header className="page-header">
          <Image
            src="/me.jpg"
            alt="Portrait of me"
            width={120}
            height={120}
            priority
            className="profile-photo"
          />
          <h1 className="sr-only">Alec Cunningham</h1>
          <nav className="social-links" aria-label="Social links">
            <a
              href="https://www.linkedin.com/in/aleccunningham/"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="LinkedIn"
            >
              in
            </a>
            <a
              href="https://github.com/moosh3"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="GitHub"
            >
              gh
            </a>
            <a
              href="http://x.com/alec_c_c_"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="X"
            >
              x
            </a>
          </nav>
        </header>

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
                    </a>
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

