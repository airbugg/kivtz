package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/drift"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/airbugg/kivtz/internal/stow"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull, apply, detect drift, and push — the one daily command",
	RunE:  runSync,
}

func runSync(_ *cobra.Command, _ []string) error {
	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	cfg, _ := config.Load(config.DefaultPath(pinfo.HomeDir))
	dotfilesDir := resolveDotfilesDir(cfg, pinfo)
	if dotfilesDir == "" {
		return fmt.Errorf("no dotfiles configured — run `kivtz init <url>` first")
	}

	fmt.Println()

	online := isOnline()

	// 1. Pull if online
	if online {
		if pulled, err := gitPull(dotfilesDir); err != nil {
			fmt.Printf("  %s %v\n", warning.Render("pull:"), err)
		} else if pulled {
			fmt.Printf("  %s\n", success.Render("pulled latest"))
		}
	} else {
		fmt.Printf("  %s\n", dim.Render("offline — skipping pull"))
	}

	// 2. Apply (auto-link safe changes)
	plan := planAll(pinfo, dotfilesDir, "")
	if plan.pending > 0 {
		if err := stow.Apply(plan.entries); err != nil {
			fmt.Printf("  %s %v\n", errStyle.Render("apply error:"), err)
		} else {
			fmt.Printf("  %s\n", success.Render(fmt.Sprintf("linked %d configs", plan.pending)))
		}
	}

	// 3. Handle conflicts
	if plan.conflicts > 0 {
		fmt.Printf("\n  %s\n\n", warning.Render(fmt.Sprintf("%d conflicts:", plan.conflicts)))
		resolveConflicts(plan.entries)
	}

	// 4. Detect drift
	ignorePatterns, _ := drift.ParseIgnoreFile(filepath.Join(dotfilesDir, ".syncignore"))
	var allDrift []drift.Entry
	for _, group := range pinfo.Groups() {
		groupDir := filepath.Join(dotfilesDir, group)
		if _, err := os.Stat(groupDir); os.IsNotExist(err) {
			continue
		}
		d, err := drift.Detect(groupDir, pinfo.HomeDir, ignorePatterns)
		if err != nil {
			continue
		}
		allDrift = append(allDrift, d...)
	}

	if len(allDrift) > 0 {
		fmt.Printf("\n  %s\n", warning.Render(fmt.Sprintf("%d drifted files:", len(allDrift))))
		for _, d := range allDrift {
			kind := "overwritten"
			if d.Kind == drift.New {
				kind = "new"
			}
			rel, _ := filepath.Rel(pinfo.HomeDir, d.Path)
			fmt.Printf("    %s  [%s] %s\n", dim.Render(kind), d.Package, rel)
		}
		fmt.Println()
	}

	// 5. Push if there are local changes
	status, _ := gitRepoStatus(dotfilesDir)
	if !status.clean {
		msg := generateCommitMessage(dotfilesDir)
		if err := gitCommitAndPush(dotfilesDir, msg, online); err != nil {
			fmt.Printf("  %s %v\n", warning.Render("save:"), err)
		} else {
			fmt.Printf("  %s %s\n", success.Render("saved:"), msg)
		}
	}

	if plan.pending == 0 && plan.conflicts == 0 && len(allDrift) == 0 && status.clean {
		fmt.Printf("  %s\n", success.Render("everything in sync"))
	}

	fmt.Println()
	return nil
}

func resolveConflicts(entries []stow.Entry) {
	reader := bufio.NewReader(os.Stdin)
	for _, e := range entries {
		if e.Action != stow.Conflict {
			continue
		}
		rel := shortPath(e.Target)
		fmt.Printf("  %s\n", bold.Render(rel))
		if e.Diff != "" {
			for _, line := range strings.Split(e.Diff, "\n") {
				switch {
				case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"):
					fmt.Printf("    %s\n", infoStyle.Render(line))
				case strings.HasPrefix(line, "+"):
					fmt.Printf("    %s\n", success.Render(line))
				case strings.HasPrefix(line, "-"):
					fmt.Printf("    %s\n", errStyle.Render(line))
				default:
					fmt.Printf("    %s\n", line)
				}
			}
		}
		fmt.Printf("  %s ", dim.Render("[a]ccept  [s]kip  ?"))
		answer, _ := reader.ReadString('\n')
		switch strings.TrimSpace(answer) {
		case "a", "A":
			e.Action = stow.Link
			if err := stow.Apply([]stow.Entry{e}); err != nil {
				fmt.Printf("  %s %v\n", errStyle.Render("error:"), err)
			} else {
				fmt.Printf("  %s\n\n", success.Render("accepted"))
			}
		default:
			fmt.Printf("  %s\n\n", dim.Render("skipped"))
		}
	}
}

func gitPull(dir string) (bool, error) {
	before, _ := runGit(dir, "rev-parse", "HEAD")
	if _, err := runGit(dir, "pull", "--ff-only"); err != nil {
		return false, err
	}
	after, _ := runGit(dir, "rev-parse", "HEAD")
	return strings.TrimSpace(before) != strings.TrimSpace(after), nil
}

func gitCommitAndPush(dir, msg string, online bool) error {
	if _, err := runGit(dir, "add", "--all"); err != nil {
		return fmt.Errorf("staging: %w", err)
	}
	if _, err := runGit(dir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	if online {
		runGit(dir, "push") // best-effort
	}
	return nil
}

func generateCommitMessage(dir string) string {
	out, err := runGit(dir, "status", "--porcelain")
	if err != nil {
		return "update configs"
	}
	pkgs := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(line) < 3 {
			continue
		}
		file := strings.TrimSpace(line[2:])
		parts := strings.SplitN(filepath.ToSlash(file), "/", 2)
		pkgs[parts[0]] = true
	}
	names := make([]string, 0, len(pkgs))
	for p := range pkgs {
		names = append(names, p)
	}
	if len(names) == 0 {
		return "update configs"
	}
	return "update " + strings.Join(names, ", ")
}

func shortPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, path); err == nil {
			return "~/" + rel
		}
	}
	return path
}
