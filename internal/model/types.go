package model

import "time"

// Repo represents a tracked repository.
type Repo struct {
	ID       int64
	Owner    string
	Name     string
	FullName string // "owner/name"
	Packages map[string]string // registry -> package name (e.g., "npm" -> "@owner/pkg")
	AddedAt  time.Time
}

// Snapshot captures point-in-time counts for a repo.
type Snapshot struct {
	ID        int64
	RepoID    int64
	Stars     int
	Forks     int
	OpenPRs   int
	TakenAt   time.Time
}

// EventKind categorizes positive events.
type EventKind string

const (
	EventStar             EventKind = "star"
	EventFork             EventKind = "fork"
	EventFirstTimePR      EventKind = "first_time_pr"
	EventMergedPR         EventKind = "merged_pr"
	EventGratitudeComment EventKind = "gratitude_comment"
	EventSponsor          EventKind = "sponsor"
	EventNotableStargazer EventKind = "notable_stargazer"
	EventDownloadSpike    EventKind = "download_spike"
	EventHNMention        EventKind = "hn_mention"
	EventRelease          EventKind = "release_adoption"
	EventMilestone        EventKind = "milestone"
)

// Event is a single positive signal from any source.
type Event struct {
	ID        string
	RepoID    int64
	Kind      EventKind
	Title     string
	Body      string
	URL       string
	Actor     string // username or display name of the person involved
	Meta      map[string]string // flexible key-value metadata
	OccuredAt time.Time
	CreatedAt time.Time
}

// MilestoneKind identifies what metric hit a milestone.
type MilestoneKind string

const (
	MilestoneStars        MilestoneKind = "stars"
	MilestoneForks        MilestoneKind = "forks"
	MilestoneContributors MilestoneKind = "contributors"
	MilestoneDownloads    MilestoneKind = "downloads"
)

// Milestone records a threshold crossing.
type Milestone struct {
	ID           int64
	RepoID       int64
	Kind         MilestoneKind
	Threshold    int
	ActualValue  int
	CelebratedAt time.Time
}

// StarThresholds defines the milestone progression for stars.
var StarThresholds = []int{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000}

// ForkThresholds defines milestone progression for forks.
var ForkThresholds = []int{5, 10, 25, 50, 100, 250, 500, 1000, 5000}

// ContributorThresholds defines milestone progression for unique contributors.
var ContributorThresholds = []int{1, 5, 10, 25, 50, 100, 250, 500}

// DownloadThresholds defines milestone progression for downloads.
var DownloadThresholds = []int{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000}

// Session records when the user last ran gh flair.
type Session struct {
	ID      int64
	RanAt   time.Time
}

// Streak tracks consecutive activity for a repo+metric.
type Streak struct {
	ID          int64
	RepoID      int64
	Metric      string // "stars", "contributors", "downloads"
	CurrentDays int
	BestDays    int
	LastActive  time.Time
}

// SavedMoment is a user-saved event for the wall of love.
type SavedMoment struct {
	ID       int64
	EventID  string
	RepoID   int64
	Kind     EventKind
	Title    string
	Body     string
	Actor    string
	URL      string
	SavedAt  time.Time
}

// HighlightReel is the assembled output for a single gh flair run.
type HighlightReel struct {
	Since        time.Time
	RepoSummaries []RepoSummary
	Milestones   []MilestoneCelebration
	Streaks      []Streak
}

// RepoSummary aggregates all positive signals for one repo.
type RepoSummary struct {
	Repo             Repo
	StarsDelta       int
	StarsTotal       int
	ForksDelta       int
	ForksTotal       int
	NewContributors  []Event
	GratitudeComments []Event
	NotableStargazers []Event
	SponsorEvents    []Event
	DownloadCount    int
	DownloadDelta    float64 // percentage change
	DownloadRegistry string  // "npm", "pypi", "crates"
	HNMentions       []Event
	ReleaseEvents    []Event
}

// MilestoneCelebration pairs a milestone with display context.
type MilestoneCelebration struct {
	Milestone      Milestone
	RepoFullName   string
	PriorMilestone *Milestone // the previous milestone in this category, if any
	Percentile     string     // e.g., "top 0.01% of GitHub projects"
}

// NotableUser holds info about a high-profile GitHub user.
type NotableUser struct {
	Login     string
	Name      string
	Followers int
	Company   string
	Location  string
}
