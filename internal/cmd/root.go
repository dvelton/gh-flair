package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dvelton/gh-flair/internal/analyzer"
	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/fetcher"
	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/presenter"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var (
	flagQuiet bool
	flagRepo  string
	flagSince string
)

var rootCmd = &cobra.Command{
	Use:   "gh-flair",
	Short: "Your repos' highlight reel.",
	RunE:  runRoot,
}

func init() {
	rootCmd.Flags().BoolVarP(&flagQuiet, "quiet", "q", false, "compact one-liner output")
	rootCmd.Flags().StringVarP(&flagRepo, "repo", "r", "", "filter to a single repo (owner/name)")
	rootCmd.Flags().StringVar(&flagSince, "since", "", "look back period: 7d, 30d, 90d, 1y (default: since last run, or 30d on first run)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(wallCmd)
	rootCmd.AddCommand(streakCmd)
	rootCmd.AddCommand(recapCmd)
	rootCmd.AddCommand(shareCmd)
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := store.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	token, err := getGHToken()
	if err != nil {
		return fmt.Errorf("get GitHub token: %w", err)
	}

	lastSession, err := st.GetLastSession()
	if err != nil {
		return fmt.Errorf("get last session: %w", err)
	}
	var since time.Time
	if flagSince != "" {
		d, err := parseDuration(flagSince)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: %w", flagSince, err)
		}
		since = time.Now().Add(-d)
	} else if lastSession == nil {
		// First run: look back 30 days to give a rich initial experience
		since = time.Now().Add(-30 * 24 * time.Hour)
	} else {
		since = lastSession.RanAt
	}

	repos, err := resolveRepos(cfg, st)
	if err != nil {
		return fmt.Errorf("resolve repos: %w", err)
	}

	if flagRepo != "" {
		var filtered []model.Repo
		for _, r := range repos {
			if r.FullName == flagRepo {
				filtered = append(filtered, r)
			}
		}
		repos = filtered
	}

	if len(repos) == 0 {
		fmt.Println("No repos configured yet.")
		fmt.Print("Run setup now? [Y/n] ")
		var answer string
		fmt.Scanln(&answer)
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			return runInit(cmd, nil)
		}
		return nil
	}

	quiet := flagQuiet || cfg.Settings.Quiet

	ctx := context.Background()
	allEvents := fetchAllEvents(ctx, token, repos, since)

	annotateStargazers(ctx, token, allEvents, cfg.Settings.NotableThreshold)

	if err := st.SaveEvents(allEvents); err != nil {
		log.Printf("warning: save events: %v", err)
	}

	an := analyzer.New(st)
	reel, err := an.BuildHighlightReel(repos, allEvents, since)
	if err != nil {
		return fmt.Errorf("build highlight reel: %w", err)
	}

	// Save snapshots AFTER building the reel so delta calculations used the previous snapshot.
	saveSnapshots(st, reel.RepoSummaries)

	sess := &model.Session{RanAt: time.Now().UTC()}
	if err := st.SaveSession(sess); err != nil {
		log.Printf("warning: save session: %v", err)
	}

	var output string
	if quiet {
		output = presenter.RenderQuiet(reel)
	} else if reelIsEmpty(reel) {
		output = renderNothingNew(reel)
	} else {
		output = presenter.RenderHighlightReel(reel)
	}
	fmt.Println(output)
	return nil
}

func getGHToken() (string, error) {
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return strings.TrimSpace(token), nil
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return strings.TrimSpace(token), nil
	}
	// Read token directly from gh CLI config file
	cfgDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home dir: %w", err)
	}
	for _, path := range []string{
		cfgDir + "/.config/gh/hosts.yml",
		cfgDir + "/.config/gh/hosts.yaml",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Simple parse: look for oauth_token line
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "oauth_token:") {
				token := strings.TrimSpace(strings.TrimPrefix(trimmed, "oauth_token:"))
				if token != "" {
					return token, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no GitHub token found: set GH_TOKEN or run 'gh auth login'")
}

func resolveRepos(cfg *config.Config, st *store.Store) ([]model.Repo, error) {
	var repos []model.Repo
	for _, rc := range cfg.Repos {
		existing, err := st.GetRepo(rc.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			repos = append(repos, *existing)
			continue
		}
		parts := strings.SplitN(rc.Name, "/", 2)
		if len(parts) != 2 {
			log.Printf("warning: invalid repo name %q, skipping", rc.Name)
			continue
		}
		pkgs := rc.Packages
		if pkgs == nil {
			pkgs = map[string]string{}
		}
		r := &model.Repo{
			Owner:    parts[0],
			Name:     parts[1],
			FullName: rc.Name,
			Packages: pkgs,
			AddedAt:  time.Now().UTC(),
		}
		if err := st.AddRepo(r); err != nil {
			return nil, err
		}
		repos = append(repos, *r)
	}
	return repos, nil
}

func fetchAllEvents(ctx context.Context, token string, repos []model.Repo, since time.Time) []model.Event {
	fetchers := []fetcher.Fetcher{
		fetcher.NewStarsFetcher(token),
		fetcher.NewEventsFetcher(token),
		fetcher.NewCommentsFetcher(token),
		fetcher.NewSponsorsFetcher(token),
		fetcher.NewNpmFetcher(),
		fetcher.NewPyPIFetcher(),
		fetcher.NewCratesFetcher(),
		fetcher.NewHNFetcher(),
	}

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		allEvents []model.Event
	)

	for _, f := range fetchers {
		wg.Add(1)
		go func(f fetcher.Fetcher) {
			defer wg.Done()
			events, err := f.Fetch(ctx, repos, since)
			if err != nil {
				log.Printf("warning: fetcher %T: %v", f, err)
				return
			}
			mu.Lock()
			allEvents = append(allEvents, events...)
			mu.Unlock()
		}(f)
	}
	wg.Wait()
	return allEvents
}

// annotateStargazers looks up follower counts for star event actors and
// sets Meta["followers"] on events that meet the threshold. This is required
// before BuildHighlightReel calls FilterNotableStargazers.
func annotateStargazers(ctx context.Context, token string, events []model.Event, threshold int) {
	uf := fetcher.NewUserFetcher()
	for i := range events {
		if events[i].Kind != model.EventStar {
			continue
		}
		user, err := uf.FetchUser(ctx, token, events[i].Actor)
		if err != nil || user.Followers < threshold {
			continue
		}
		if events[i].Meta == nil {
			events[i].Meta = make(map[string]string)
		}
		events[i].Meta["followers"] = fmt.Sprintf("%d", user.Followers)
	}
}

func saveSnapshots(st *store.Store, summaries []model.RepoSummary) {
	for _, s := range summaries {
		snap := &model.Snapshot{
			RepoID:  s.Repo.ID,
			Stars:   s.StarsTotal,
			Forks:   s.ForksTotal,
			OpenPRs: 0,
			TakenAt: time.Now().UTC(),
		}
		if err := st.SaveSnapshot(snap); err != nil {
			log.Printf("warning: save snapshot for %s: %v", s.Repo.FullName, err)
		}
	}
}

func reelIsEmpty(reel *model.HighlightReel) bool {
	if len(reel.Milestones) > 0 {
		return false
	}
	for _, st := range reel.Streaks {
		if st.CurrentDays > 0 {
			return false
		}
	}
	for _, s := range reel.RepoSummaries {
		if s.StarsDelta > 0 || s.ForksDelta > 0 ||
			len(s.NewContributors) > 0 || len(s.GratitudeComments) > 0 ||
			len(s.NotableStargazers) > 0 || len(s.SponsorEvents) > 0 ||
			s.DownloadCount > 0 || len(s.HNMentions) > 0 ||
			len(s.ReleaseEvents) > 0 {
			return false
		}
	}
	return true
}

func renderNothingNew(reel *model.HighlightReel) string {
	totalStars := 0
	for _, s := range reel.RepoSummaries {
		totalStars += s.StarsTotal
	}
	msg := "✦ Nothing new since last check"
	if totalStars > 0 {
		msg += fmt.Sprintf(" — %d total stars across your repos", totalStars)
	}
	return msg + ". Keep shipping!"
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	suffix := s[len(s)-1:]
	numStr := s[:len(s)-1]

	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", numStr)
	}
	if n <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	switch suffix {
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	case "w":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(n) * 30 * 24 * time.Hour, nil
	case "y":
		return time.Duration(n) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown suffix %q (use d, w, m, or y)", suffix)
	}
}
