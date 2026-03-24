package cmd

import (
	"fmt"
	"time"

	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save <event-id>",
	Short: "Save a moment to your wall of love.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSave,
}

func runSave(cmd *cobra.Command, args []string) error {
	eventID := args[0]

	st, err := store.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	event, err := st.GetEventByID(eventID)
	if err != nil {
		return fmt.Errorf("look up event: %w", err)
	}
	if event == nil {
		return fmt.Errorf("event %q not found", eventID)
	}

	moment := &model.SavedMoment{
		EventID: event.ID,
		RepoID:  event.RepoID,
		Kind:    event.Kind,
		Title:   event.Title,
		Body:    event.Body,
		Actor:   event.Actor,
		URL:     event.URL,
		SavedAt: time.Now().UTC(),
	}
	if err := st.SaveMoment(moment); err != nil {
		return fmt.Errorf("save moment: %w", err)
	}

	fmt.Printf("✓ Saved moment #%d — %s\n", moment.ID, event.Title)
	return nil
}
