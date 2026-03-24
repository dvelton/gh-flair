package analyzer

import (
	"time"

	"github.com/dvelton/gh-flair/internal/store"
)

// CalcStarDelta returns how many stars have been gained since `since` by
// comparing currentStars against the most recent snapshot taken before that
// time. Returns 0 if no prior snapshot exists.
func CalcStarDelta(s *store.Store, repoID int64, currentStars int, since time.Time) int {
	snap, err := s.GetSnapshotBefore(repoID, since)
	if err != nil || snap == nil {
		return 0
	}
	delta := currentStars - snap.Stars
	if delta < 0 {
		return 0
	}
	return delta
}

// CalcForkDelta returns how many forks have been gained since `since` by
// comparing currentForks against the most recent snapshot taken before that
// time. Returns 0 if no prior snapshot exists.
func CalcForkDelta(s *store.Store, repoID int64, currentForks int, since time.Time) int {
	snap, err := s.GetSnapshotBefore(repoID, since)
	if err != nil || snap == nil {
		return 0
	}
	delta := currentForks - snap.Forks
	if delta < 0 {
		return 0
	}
	return delta
}

// CalcDownloadDelta returns the percentage change from previousCount to
// currentCount. Returns 0 if previousCount is 0 to avoid division by zero.
func CalcDownloadDelta(previousCount, currentCount int) float64 {
	if previousCount == 0 {
		return 0
	}
	return float64(currentCount-previousCount) / float64(previousCount) * 100
}
