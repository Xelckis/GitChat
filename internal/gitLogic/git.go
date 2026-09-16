package gitLogic

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

type Commit struct {
	Hash        string
	AuthorName  string
	AuthorEmail string
	When        time.Time
	Committer   string
	Message     string
}

type User struct {
	Name  string
	Email string
}

func ReadHistory(limit, offset int) ([]Commit, error) {
	r, err := git.PlainOpen("./repoDB")
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("repository does not exist")
		}
		return nil, fmt.Errorf("error opening repository: %w", err)
	}

	ref, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("error getting repository head: %w", err)
	}

	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Order: git.LogOrderCommitterTime})
	if err != nil {
		return nil, fmt.Errorf("error retrieving Commit history: %w", err)
	}
	defer cIter.Close()

	var commits []Commit
	var skipped int

	for {
		c, err := cIter.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("error iterating over Commit history: %w", err)
		}

		if skipped < offset {
			skipped++
			continue
		}

		message := Commit{
			Hash:        c.Hash.String(),
			AuthorName:  c.Author.Name,
			AuthorEmail: c.Author.Email,
			When:        c.Author.When,
			Committer:   c.Committer.String(),
			Message:     c.Message,
		}

		commits = append(commits, message)

		if len(commits) == limit {
			break
		}
	}

	return commits, nil

}

func SendMessage(message, name, email string) error {
	r, err := git.PlainOpen("./repoDB")
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return fmt.Errorf("repository does not exist")
		}
		return fmt.Errorf("error opening repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %w", err)
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
		AllowEmptyCommits: true,
	})
	if err != nil {
		return fmt.Errorf("error committing changes: %w", err)
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		log.Println("warning: failed sending message to remote repo:", err)
	}

	return nil
}

func getGitConfig() (string, string, error) {
	nameCmd := exec.Command("git", "config", "user.name")
	nameByte, err := nameCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error executing command or key does not exist: %w", err)
	}

	emailCmd := exec.Command("git", "config", "user.email")
	emailByte, err := emailCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error executing command or key does not exist: %w", err)
	}

	name := strings.TrimSpace(string(nameByte))
	if name == "" {
		return "", "", fmt.Errorf("user.name is empty")
	}

	email := strings.TrimSpace(string(emailByte))
	if email == "" {
		return "", "", fmt.Errorf("user.email is empty")
	}

	return name, email, nil

}

func SetUserConfig() (User, error) {
	name, email, err := getGitConfig()
	if err != nil {
		return User{}, err
	}
	return User{Name: name, Email: email}, nil
}

func StartSyncLoop(repoPath string) {
	go func() {
		for {
			time.Sleep(5 * time.Second)

			r, err := git.PlainOpen(repoPath)
			if err != nil {
				continue
			}

			w, err := r.Worktree()
			if err != nil {
				continue
			}

			err = w.Pull(&git.PullOptions{
				RemoteName: "origin",
			})

			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				log.Println("error syncing with remote repo:", err)
			}
		}
	}()
}
