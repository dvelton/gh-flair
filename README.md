# gh flair

Your repos' highlight reel.

A `gh` CLI extension that surfaces the good news from your open source projects — new stars, grateful users, first-time contributors, sponsor events, download milestones, and Hacker News mentions. The signals that make maintaining worth it, pulled out of the noise and presented when you open your terminal.

## Install

```
gh extension install dvelton/gh-flair
```

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated.

## Usage

```
gh flair
```

That's it. Run it, get a highlight reel of good things that happened since you last checked:

```
  ✦ gh flair — since yesterday 6:42 AM

  ★ Stars
    yourname/cool-project    +12 stars (→ 488, 12 away from 500!)
    yourname/tiny-lib         +3 stars (→ 47)
    🐋 @sindresorhus starred cool-project (72K followers)

  🎉 Milestones
    yourname/cool-project crossed 475 stars

  👋 New Contributors
    @alice-dev opened their first PR to cool-project (#234)

  💬 Kind Words
    @user123 on cool-project#230:
      "This library saved me two days of work. Thank you."

  💰 Sponsors
    @generous-corp started sponsoring you ($10/mo)

  📦 Downloads
    cool-project (npm): 14,201 this week (+22%)

  📰 Hacker News
    cool-project was posted to HN — 342 points, 89 comments
    https://news.ycombinator.com/item?id=12345678

  🔥 Streaks
    cool-project: starred 34 days in a row
```

## Setup

```
gh flair init
```

Interactive setup that auto-discovers your repos and lets you pick which ones to track. Supports individual numbers (`1,3,5`), ranges (`1-10`), or `all` to track everything. Optionally map repos to package registries (npm, PyPI, crates.io) for download tracking.

On first run after setup, `gh flair` automatically looks back 30 days to give you a rich initial highlight reel.

## Commands

| Command | What it does |
|---|---|
| `gh flair` | Morning highlight reel (default) |
| `gh flair init` | Interactive repo setup |
| `gh flair save <id>` | Save an event to your wall of love |
| `gh flair wall` | Interactive browse of saved wins |
| `gh flair streak` | View activity streaks |
| `gh flair recap --year 2025` | Yearly/monthly highlight summary |
| `gh flair share --repo owner/name --milestone "1K stars"` | Generate a shareable milestone card |
| `gh flair --quiet` | One-liner for shell startup integration |

## Flags

| Flag | Description |
|---|---|
| `--quiet`, `-q` | Compact one-line output |
| `--repo`, `-r` | Filter to a single repo |
| `--since` | Look back period: `7d`, `30d`, `90d`, `1y` |

Examples:

```bash
gh flair --since 90d              # highlights from the last 90 days
gh flair --since 1y --repo owner/repo  # one repo, one year
```

## Shell Integration

Add to your `.zshrc` or `.bashrc` for a daily dopamine hit on terminal open:

```bash
gh flair --quiet
```

## Wall of Love

Save the moments that matter:

```
gh flair save evt_abc123
```

Browse your saved wins interactively:

```
gh flair wall
```

Export as markdown for READMEs, sponsor pitches, or grant applications:

```
gh flair wall --format markdown > wall-of-love.md
```

## What It Tracks

| Signal | Source |
|---|---|
| New stars | GitHub API |
| Star milestones | 10, 25, 50, 100, 250, 500, 1K, 2.5K, 5K, 10K, 25K, 50K, 100K |
| First-time contributors | GitHub PR events |
| Gratitude comments | Keyword-filtered issue/PR comments |
| Notable stargazers | High-follower users who star your repo |
| New sponsors | GitHub Sponsors |
| Download growth | npm, PyPI, crates.io |
| Forks | GitHub Events API |
| HN mentions | Hacker News Algolia API |

## What It Does Not Track

Bug reports, feature requests, security alerts, CI failures, issue backlogs, or anything that creates pressure. This is a highlight reel, not a notification inbox.

## How It Works

- Runs locally as a `gh` CLI extension (single Go binary)
- Auth via your existing `gh auth` token — no additional setup
- All state stored in a local SQLite database at `~/.config/gh-flair/flair.db`
- No server, no account, no telemetry, no data leaves your machine
- Config lives at `~/.config/gh-flair/config.yaml`

## Configuration

```yaml
repos:
  - name: owner/repo-name
    packages:
      npm: "@owner/package-name"
      pypi: package-name
  - name: owner/other-repo

settings:
  streaks: true
  notable_threshold: 1000
  quiet: false
```

## Design Principles

1. **News feed, not dashboard.** Moments, not charts.
2. **Celebrate, don't analyze.** No line charts, no funnels.
3. **No bad news.** Highlight reel only.
4. **Deltas over totals.** "+12 since yesterday" over "4,847 total."
5. **Pull-based.** You check when you want the mood boost.
6. **Fast.** Output in under 2 seconds.
7. **Never empty.** Always shows something positive.

## License

MIT
