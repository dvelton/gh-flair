package analyzer

import (
	"fmt"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/store"
)

// Analyzer assembles raw events and store snapshots into a HighlightReel.
type Analyzer struct {
	store *store.Store
}

// New creates an Analyzer backed by the given store.
func New(s *store.Store) *Analyzer {
	return &Analyzer{store: s}
}

// BuildHighlightReel is the main entry point. It groups events by repo,
// calculates star/fork deltas from snapshots, detects milestones, updates
// streaks, and assembles the full HighlightReel.
func (a *Analyzer) BuildHighlightReel(repos []model.Repo, events []model.Event, since time.Time) (*model.HighlightReel, error) {
	// Index repos by ID for quick lookup.
	repoByID := make(map[int64]model.Repo, len(repos))
	for _, r := range repos {
		repoByID[r.ID] = r
	}

	// Group events by repo ID.
	eventsByRepo := make(map[int64][]model.Event)
	for _, e := range events {
		eventsByRepo[e.RepoID] = append(eventsByRepo[e.RepoID], e)
	}

	reel := &model.HighlightReel{Since: since}

	for _, repo := range repos {
		repoEvents := eventsByRepo[repo.ID]

		summary, err := a.buildRepoSummary(repo, repoEvents, since)
		if err != nil {
			return nil, err
		}

		// Star milestones.
		starMilestones, err := DetectMilestones(a.store, repo.ID, model.MilestoneStars, summary.StarsTotal, model.StarThresholds)
		if err != nil {
			return nil, err
		}
		for _, mc := range starMilestones {
			mc.RepoFullName = repo.FullName
			reel.Milestones = append(reel.Milestones, mc)
		}

		// Fork milestones.
		forkMilestones, err := DetectMilestones(a.store, repo.ID, model.MilestoneForks, summary.ForksTotal, model.ForkThresholds)
		if err != nil {
			return nil, err
		}
		for _, mc := range forkMilestones {
			mc.RepoFullName = repo.FullName
			reel.Milestones = append(reel.Milestones, mc)
		}

		// Streaks.
		streaks, err := UpdateStreaks(a.store, repo.ID, repoEvents)
		if err != nil {
			return nil, err
		}
		reel.Streaks = append(reel.Streaks, streaks...)

		reel.RepoSummaries = append(reel.RepoSummaries, summary)
	}

	return reel, nil
}

// buildRepoSummary assembles a RepoSummary for one repo from its events.
func (a *Analyzer) buildRepoSummary(repo model.Repo, events []model.Event, since time.Time) (model.RepoSummary, error) {
	summary := model.RepoSummary{Repo: repo}

	// Collect current star/fork counts from the latest snapshot.
	latest, err := a.store.GetLatestSnapshot(repo.ID)
	if err != nil {
		return summary, err
	}
	if latest != nil {
		summary.StarsTotal = latest.Stars
		summary.ForksTotal = latest.Forks
	}

	// Deltas against the snapshot just before `since`.
	summary.StarsDelta = CalcStarDelta(a.store, repo.ID, summary.StarsTotal, since)
	summary.ForksDelta = CalcForkDelta(a.store, repo.ID, summary.ForksTotal, since)

	// Bucket events by kind.
	const notableThreshold = 500
	for _, e := range events {
		switch e.Kind {
		case model.EventFirstTimePR:
			summary.NewContributors = append(summary.NewContributors, e)
		case model.EventGratitudeComment:
			// Raw comments are filtered below after collecting all.
		case model.EventSponsor:
			summary.SponsorEvents = append(summary.SponsorEvents, e)
		case model.EventHNMention:
			summary.HNMentions = append(summary.HNMentions, e)
		case model.EventRelease:
			summary.ReleaseEvents = append(summary.ReleaseEvents, e)
		case model.EventDownloadSpike:
			if e.Meta != nil {
				if v, ok := e.Meta["download_count"]; ok {
					var n int
					if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
						summary.DownloadCount = n
					}
				}
				if reg, ok := e.Meta["registry"]; ok {
					summary.DownloadRegistry = reg
				}
			}
		}
	}

	// Filter gratitude comments from all comment events.
	summary.GratitudeComments = FilterGratitude(events)

	// Detect notable stargazers from star events.
	summary.NotableStargazers = FilterNotableStargazers(events, notableThreshold)

	return summary, nil
}
