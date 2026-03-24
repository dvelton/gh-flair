package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/presenter"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var (
	recapFlagYear  int
	recapFlagMonth int
)

var recapCmd = &cobra.Command{
	Use:   "recap",
	Short: "Summarise a past month or year.",
	RunE:  runRecap,
}

func init() {
	recapCmd.Flags().IntVar(&recapFlagYear, "year", 0, "year to recap (default: current year)")
	recapCmd.Flags().IntVar(&recapFlagMonth, "month", 0, "month to recap (1-12; omit for full year)")
}

func runRecap(cmd *cobra.Command, args []string) error {
	st, err := store.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	now := time.Now()
	year := recapFlagYear
	if year == 0 {
		year = now.Year()
	}
	month := recapFlagMonth

	var since, until time.Time
	var period string
	if month > 0 {
		since = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		until = since.AddDate(0, 1, 0)
		period = since.Format("January 2006")
	} else {
		since = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		until = since.AddDate(1, 0, 0)
		period = strconv.Itoa(year)
	}

	repos, err := st.ListRepos()
	if err != nil {
		return fmt.Errorf("list repos: %w", err)
	}
	if len(repos) == 0 {
		fmt.Println("No repos tracked yet. Run 'gh flair init' first.")
		return nil
	}

	var summaries []model.RepoSummary
	for _, repo := range repos {
		events, err := st.GetEventsSince(repo.ID, since, nil)
		if err != nil {
			return fmt.Errorf("get events for %s: %w", repo.FullName, err)
		}

		// Trim events after the period end.
		var periodEvents []model.Event
		for _, e := range events {
			if e.OccuredAt.Before(until) {
				periodEvents = append(periodEvents, e)
			}
		}

		summary := model.RepoSummary{Repo: repo}

		// Get total stars from latest snapshot.
		if snap, err := st.GetLatestSnapshot(repo.ID); err == nil && snap != nil {
			summary.StarsTotal = snap.Stars
			summary.ForksTotal = snap.Forks
		}

		for _, e := range periodEvents {
			switch e.Kind {
			case model.EventStar:
				summary.StarsDelta++
			case model.EventFork:
				summary.ForksDelta++
			case model.EventFirstTimePR:
				summary.NewContributors = append(summary.NewContributors, e)
			case model.EventSponsor:
				summary.SponsorEvents = append(summary.SponsorEvents, e)
			case model.EventDownloadSpike:
				if e.Meta != nil {
					if v, ok := e.Meta["downloads"]; ok {
						var n int
						if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > summary.DownloadCount {
							summary.DownloadCount = n
						}
					}
					if reg, ok := e.Meta["registry"]; ok {
						summary.DownloadRegistry = reg
					}
				}
			}
		}

		summaries = append(summaries, summary)
	}

	fmt.Println(presenter.RenderRecap(summaries, period))
	return nil
}
