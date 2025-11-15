# Garden Setup Guide

A minimalistic personal homepage showcasing your online activity from Letterboxd, Goodreads, and GitHub.

## Prerequisites

- Node.js 18.x or higher
- npm or yarn

## Installation

1. Install dependencies:

```bash
npm install
```

2. Configure environment variables:

Create a `.env.local` file in the root directory with your credentials:

```bash
cp env.template .env.local
```

Then edit `.env.local` with your information:

- `NEXT_PUBLIC_LETTERBOXD_USERNAME`: Your Letterboxd username
- `NEXT_PUBLIC_GOODREADS_USER_ID`: Your Goodreads user ID (found in your profile URL)
- `NEXT_PUBLIC_GITHUB_USERNAME`: Your GitHub username
- `GITHUB_TOKEN`: (Optional) GitHub personal access token for higher API rate limits

## Development

Run the development server:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

## Building for Production

Build a static export:

```bash
npm run build
```

The static files will be generated in the `out/` directory.

## Deployment

### Vercel (Recommended)

The project is configured for static export and will work with Vercel's static hosting.

**Option 1: Using Vercel CLI**

1. Install Vercel CLI:
```bash
npm i -g vercel
```

2. Deploy:
```bash
vercel
```

3. Set environment variables in Vercel dashboard under Settings → Environment Variables

**Option 2: Using Vercel Dashboard**

1. Push your code to GitHub
2. Import the repository in Vercel
3. Vercel will auto-detect the configuration from `vercel.json`
4. Add environment variables in Project Settings → Environment Variables
5. Deploy

**Important:** The `vercel.json` is configured to use the static export from the `out/` directory. Environment variables must be set before build time as they're baked into the static files.

### Netlify

1. Build command: `npm run build`
2. Publish directory: `out`
3. Add environment variables in Site settings → Environment variables

### GitHub Pages

1. Push your code to GitHub
2. Go to Settings → Pages
3. Set up GitHub Actions for deployment
4. Add environment variables as GitHub Secrets

## Customization

### Update About Section

Edit the about section in `pages/index.tsx`:

```typescript
<section className="about">
  <h2>About</h2>
  <p>Your content here...</p>
</section>
```

### Styling

Modify `styles/globals.css` to adjust colors, fonts, and spacing. The current theme uses:
- White background (`#ffffff`)
- Green accents (`#2d5c2d`) for links and highlights
- System font stack for clean typography

### Activity Sources

The activity feeds are configured in `lib/api.ts`. You can:
- Adjust the number of items displayed (currently 5 each)
- Modify date formatting
- Change which GitHub event types are shown

## Troubleshooting

### RSS Feeds Not Loading

- Verify your usernames/IDs are correct in `.env.local`
- Check that your profiles are public
- Letterboxd and Goodreads RSS feeds require public profiles

### GitHub API Rate Limits

Without a token, GitHub API allows 60 requests/hour. With a token, this increases to 5,000 requests/hour.

Create a token at: https://github.com/settings/tokens (no special scopes needed for public data)

## License

MIT

