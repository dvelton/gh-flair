package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure repos to track.",
	RunE:  runInit,
}

type ghRepoItem struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func runInit(cmd *cobra.Command, args []string) error {
	out, err := exec.Command("gh", "repo", "list", "--source", "--json", "owner,name", "--limit", "100").Output()
	if err != nil {
		return fmt.Errorf("gh repo list: %w", err)
	}

	var items []ghRepoItem
	if err := json.Unmarshal(out, &items); err != nil {
		return fmt.Errorf("parse repo list: %w", err)
	}
	if len(items) == 0 {
		fmt.Println("No source repos found.")
		return nil
	}

	fmt.Println("Your repos:")
	for i, item := range items {
		fmt.Printf("  %d) %s/%s\n", i+1, item.Owner.Login, item.Name)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nEnter numbers to track (comma-separated, e.g. 1,3,5): ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read selection: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		fmt.Println("No repos selected.")
		return nil
	}

	var selected []ghRepoItem
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(items) {
			fmt.Printf("  Skipping invalid selection %q\n", part)
			continue
		}
		selected = append(selected, items[n-1])
	}
	if len(selected) == 0 {
		fmt.Println("No valid repos selected.")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := store.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	existingByName := make(map[string]bool)
	for _, rc := range cfg.Repos {
		existingByName[rc.Name] = true
	}

	for _, item := range selected {
		fullName := item.Owner.Login + "/" + item.Name

		pkgs := map[string]string{}
		fmt.Printf("\nPackage registries for %s (e.g. npm:@scope/pkg, pypi:name, crates:name)\nPress Enter to skip: ", fullName)
		pkgLine, _ := reader.ReadString('\n')
		pkgLine = strings.TrimSpace(pkgLine)
		if pkgLine != "" {
			for _, pair := range strings.Split(pkgLine, ",") {
				pair = strings.TrimSpace(pair)
				kv := strings.SplitN(pair, ":", 2)
				if len(kv) == 2 {
					pkgs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}

		if !existingByName[fullName] {
			cfg.Repos = append(cfg.Repos, config.RepoConfig{
				Name:     fullName,
				Packages: pkgs,
			})
		}

		existing, err := st.GetRepo(fullName)
		if err != nil {
			return err
		}
		if existing == nil {
			r := &model.Repo{
				Owner:    item.Owner.Login,
				Name:     item.Name,
				FullName: fullName,
				Packages: pkgs,
				AddedAt:  time.Now().UTC(),
			}
			if err := st.AddRepo(r); err != nil {
				return fmt.Errorf("add repo %s: %w", fullName, err)
			}
		}
		fmt.Printf("  ✓ Added %s\n", fullName)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("\nConfig saved. Run 'gh flair' to see your highlight reel.\n")
	return nil
}
