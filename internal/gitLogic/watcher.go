package gitLogic

import (
	"encoding/json"
	"log"
	"path/filepath"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
)

func WatchRepo(repoPath string, broadcast func([]byte)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(repoPath, ".git", "refs", "heads")
	if err := watcher.Add(targetDir); err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		var lastHash string

		initialCommits, err := ReadHistory(1, 0)
		if err == nil && len(initialCommits) > 0 {
			lastHash = initialCommits[0].Hash
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if filepath.Ext(event.Name) == ".lock" {
					continue
				}

				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
					time.Sleep(50 * time.Millisecond)

					var newCommits []Commit
					offset := 0
					limit := 20
					found := false

					for !found {
						commits, err := ReadHistory(limit, offset)

						if err != nil || len(commits) == 0 {
							break
						}

						for _, c := range commits {
							if c.Hash == lastHash {
								found = true
								break
							}
							newCommits = append(newCommits, c)
						}

						offset += limit
					}

					if len(newCommits) > 0 {
						lastHash = newCommits[0].Hash

						for _, newCommit := range slices.Backward(newCommits) {
							payload, err := json.Marshal(newCommit)
							if err == nil {
								broadcast(payload)
							}
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("inotify error:", err)
			}
		}
	}()
	return nil
}
