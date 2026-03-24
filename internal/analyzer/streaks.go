package analyzer

import (
	"fmt"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/store"
)

// streakMetrics are the three tracked streak types.
var streakMetrics = []string{"stars", "contributors", "downloads"}

// UpdateStreaks loads existing streaks for a repo, advances or resets each one
// based on the provided events, saves the results, and returns the updated
// streak records.
//
// Streak granularity:
//   - "stars"        — consecutive calendar days with at least one star event
//   - "contributors" — consecutive ISO weeks with at least one first-time PR event
//   - "downloads"    — consecutive ISO weeks with at least one download-spike event
func UpdateStreaks(s *store.Store, repoID int64, events []model.Event) ([]model.Streak, error) {
	existing, err := s.GetStreaks(repoID)
	if err != nil {
		return nil, fmt.Errorf("analyzer: get streaks: %w", err)
	}

	// Index existing streaks by metric for easy mutation.
	byMetric := make(map[string]*model.Streak)
	for i := range existing {
		byMetric[existing[i].Metric] = &existing[i]
	}

	// Ensure every metric has a record (create zero-value if missing).
	for _, metric := range streakMetrics {
		if _, ok := byMetric[metric]; !ok {
			byMetric[metric] = &model.Streak{
				RepoID: repoID,
				Metric: metric,
			}
		}
	}

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	_, thisWeek := now.ISOWeek()
	thisYear, _ := now.ISOWeek()

	// Check activity in the provided events.
	hadStarToday := hasEventOnDay(events, model.EventStar, today)
	hadContributorThisWeek := hasEventInWeek(events, model.EventFirstTimePR, thisYear, thisWeek)
	hadDownloadThisWeek := hasEventInWeek(events, model.EventDownloadSpike, thisYear, thisWeek)

	advanceOrReset(byMetric["stars"], hadStarToday, now, true)
	advanceOrReset(byMetric["contributors"], hadContributorThisWeek, now, false)
	advanceOrReset(byMetric["downloads"], hadDownloadThisWeek, now, false)

	var updated []model.Streak
	for _, metric := range streakMetrics {
		st := byMetric[metric]
		if err := s.SaveStreak(st); err != nil {
			return nil, fmt.Errorf("analyzer: save streak %s: %w", metric, err)
		}
		updated = append(updated, *st)
	}

	return updated, nil
}

// advanceOrReset increments CurrentDays when active is true, resets to 0 (or
// 1 if newly active today) otherwise. Updates BestDays and LastActive.
//
// isDailyMetric controls whether "activity" means today vs. this calendar week.
func advanceOrReset(st *model.Streak, active bool, now time.Time, isDailyMetric bool) {
	if !active {
		// Grace: only reset if the gap is larger than one period.
		var gracePeriod time.Duration
		if isDailyMetric {
			gracePeriod = 48 * time.Hour // allow a missed day before breaking
		} else {
			gracePeriod = 14 * 24 * time.Hour // allow a missed week
		}
		if !st.LastActive.IsZero() && now.Sub(st.LastActive) > gracePeriod {
			st.CurrentDays = 0
		}
		return
	}

	// Avoid double-counting if already updated this period.
	var alreadyCounted bool
	if isDailyMetric {
		today := now.Truncate(24 * time.Hour)
		alreadyCounted = !st.LastActive.IsZero() && st.LastActive.Truncate(24*time.Hour).Equal(today)
	} else {
		_, nowWeek := now.ISOWeek()
		_, lastWeek := st.LastActive.ISOWeek()
		nowYear, _ := now.ISOWeek()
		lastYear, _ := st.LastActive.ISOWeek()
		alreadyCounted = !st.LastActive.IsZero() && nowYear == lastYear && nowWeek == lastWeek
	}

	if !alreadyCounted {
		st.CurrentDays++
	}

	if st.CurrentDays > st.BestDays {
		st.BestDays = st.CurrentDays
	}
	st.LastActive = now
}

// hasEventOnDay returns true if any event of the given kind occurred on day.
func hasEventOnDay(events []model.Event, kind model.EventKind, day time.Time) bool {
	for _, e := range events {
		if e.Kind == kind && e.OccuredAt.UTC().Truncate(24*time.Hour).Equal(day) {
			return true
		}
	}
	return false
}

// hasEventInWeek returns true if any event of the given kind occurred in the
// specified ISO year+week.
func hasEventInWeek(events []model.Event, kind model.EventKind, year, week int) bool {
	for _, e := range events {
		if e.Kind != kind {
			continue
		}
		y, w := e.OccuredAt.UTC().ISOWeek()
		if y == year && w == week {
			return true
		}
	}
	return false
}
