package analyzer

import (
	"fmt"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/store"
)

// DetectMilestones checks each threshold against currentValue for a given
// repo+kind pair. Thresholds that have already been celebrated (per the store)
// are skipped. Newly crossed thresholds are recorded in the store and returned
// as MilestoneCelebrations with prior-milestone context.
func DetectMilestones(s *store.Store, repoID int64, kind model.MilestoneKind, currentValue int, thresholds []int) ([]model.MilestoneCelebration, error) {
	// Retrieve the previously celebrated milestone for context.
	prior, err := s.GetLastMilestone(repoID, kind)
	if err != nil {
		return nil, fmt.Errorf("analyzer: get last milestone: %w", err)
	}

	var celebrations []model.MilestoneCelebration

	for _, threshold := range thresholds {
		if currentValue < threshold {
			continue
		}

		already, err := s.IsMilestoneCelebrated(repoID, kind, threshold)
		if err != nil {
			return nil, fmt.Errorf("analyzer: check milestone celebrated: %w", err)
		}
		if already {
			continue
		}

		m := model.Milestone{
			RepoID:       repoID,
			Kind:         kind,
			Threshold:    threshold,
			ActualValue:  currentValue,
			CelebratedAt: time.Now().UTC(),
		}
		if err := s.SaveMilestone(&m); err != nil {
			return nil, fmt.Errorf("analyzer: save milestone: %w", err)
		}

		celebrations = append(celebrations, model.MilestoneCelebration{
			Milestone:      m,
			PriorMilestone: prior,
		})

		// The milestone we just saved becomes the prior context for the next
		// threshold in this same run (in case multiple are crossed at once).
		copy := m
		prior = &copy
	}

	return celebrations, nil
}
