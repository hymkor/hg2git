package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/zetamatta/hg2git/com"
)

var fullAuthor = regexp.MustCompile(`\<\w+\@[\w\.]+\>\s*$`)

func author(org string) string {
	if fullAuthor.MatchString(org) {
		return org
	}
	atmarkPos := strings.IndexRune(org, '@')
	if atmarkPos >= 0 {
		return fmt.Sprintf("%s <%s>", org[:atmarkPos], org)
	}
	return fmt.Sprintf("%s <%s@localhost>", org)
}

func hgClone(src, dst string) error {
	return com.Run("hg", "clone", src, dst)
}

func gitInit() (string, error) {
	if err := com.Run("git", "init"); err != nil {
		return "", err
	}
	err := com.Run("git", "config", "--local", "core.autocrlf", "false")
	if err != nil {
		return "", err
	}
	err = com.Run("git", "commit", "-m", "zero", "--allow-empty")
	if err != nil {
		return "", err
	}
	return getCurrentGitCommit()
}

func hgUpdateC(id string) error {
	return com.Run("hg", "update", "-C", id)
}

func getHgChange(id string) ([]string, []string, error) {
	output, err := com.Quote("hg", "status", "--change", id)
	if err != nil {
		return nil, nil, err
	}
	var add, remove []string
	for _, line := range strings.Split(output, "\n") {
		if len(line) >= 2 {
			if line[0] == 'R' {
				remove = append(remove, line[2:])
			} else {
				add = append(add, line[2:])
			}
		}
	}
	return add, remove, nil
}

func gitAdd(files ...string) error {
	if len(files) <= 0 {
		return nil
	}
	args := []string{"add"}
	args = append(args, files...)
	return com.Run("git", args...)
}

func gitRemove(files ...string) error {
	if len(files) <= 0 {
		return nil
	}
	args := []string{"rm"}
	args = append(args, files...)
	return com.Run("git", args...)
}

func getCurrentGitCommit() (string, error) {
	return com.Quote("git", "log", "-n", "1", "--format=%H")
}

func gitCommit(desc string, date time.Time, user string) (string, error) {
	err := com.Run("git", "commit",
		"-m", desc,
		"--date", date.Format("Mon Jan 2 15:04:05 2006 -0700"),
		"--author", author(user),
		"--allow-empty",
		"-a")
	if err != nil {
		return "", err
	}
	return getCurrentGitCommit()
}

func hgOneCommitToGit(cs *ChangeSet, warn func(error) error) (string, error) {
	if err := hgUpdateC(cs.ChangeSetId); err != nil {
		return "", err
	}
	add, remove, err := getHgChange(cs.ChangeSetId)
	if err != nil {
		return "", err
	}
	if err := gitRemove(remove...); err != nil {
		if err = warn(err); err != nil {
			return "", err
		}
	}
	if err := gitAdd(add...); err != nil {
		if err = warn(err); err != nil {
			return "", err
		}
	}
	return gitCommit(
		fmt.Sprintf("%s\nHG: %s",
			cs.Description,
			cs.ChangeSetId,
		), cs.Date, cs.User)
}

func gitMerge(branch string) func() {
	com.Run("git", "merge", "--no-commit", "--no-edit", branch)
	return func() {
		com.Run("git", "branch", "-d", branch)
	}
}

func gitCheckout(id, newbranch string) {
	com.Run("git", "checkout", "-f", id)
	com.Run("git", "checkout", "-b", newbranch)
}

func Trace(src, dst string) error {
	var rep Repository

	err := rep.Load(src, func(err error) error {
		println(err.Error())
		return err
	})
	if err != nil {
		return err
	}

	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("%s: %w", dst, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("%s: %w", dst, err)
	}
	if err := hgClone(src, dst); err != nil {
		return err
	}

	saveDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(saveDir)
	if err := os.Chdir(dst); err != nil {
		return err
	}
	lastGitId, err := gitInit()
	if err != nil {
		return err
	}
	lastHgId := nullCommit

	branchSerial := 0
	branchName := "master"

	HgIdToGit := map[string][2]string{
		nullCommit: [2]string{lastGitId, branchName},
	}
	rep.BySerial[-1] = &ChangeSet{Serial: -1, ChangeSetId: nullCommit}

	gc := make([]func(), 0, 2)
	for serial := 0; ; serial++ {
		cs, ok := rep.BySerial[serial]
		if !ok {
			break
		}
		if len(cs.Parents) >= 1 {
			if cs.Parents[0].ChangeSetId == lastHgId {
				if len(cs.Parents) >= 2 {
					p, ok := HgIdToGit[cs.Parents[1].ChangeSetId]
					if !ok {
						return fmt.Errorf("Git-Commit for ChangeSet '%s' not found (case1)", cs.Parents[1].ChangeSetId)

					}
					gc = append(gc, gitMerge(p[1]))
				}
			} else if len(cs.Parents) >= 2 && cs.Parents[1].ChangeSetId == lastHgId {
				p, ok := HgIdToGit[cs.Parents[0].ChangeSetId]
				if !ok {
					return fmt.Errorf("Git-Commit for ChangeSet '%s' not found (case2)", cs.Parents[0].ChangeSetId)
				}
				gc = append(gc, gitMerge(p[0]))
			} else {
				// new branch
				branchSerial++
				branchName = fmt.Sprintf("fork%04d", branchSerial)
				p1, ok := HgIdToGit[cs.Parents[0].ChangeSetId]
				if !ok {
					return fmt.Errorf("Git-Commit for ChangeSet '%s' not found (case3)",
						cs.Parents[0].ChangeSetId)
				}
				gitCheckout(p1[0], branchName)
				if len(cs.Parents) >= 2 {
					gc = append(gc, func() { com.Run("git", "branch", "-d", HgIdToGit[cs.Parents[0].ChangeSetId][1]) })
					gc = append(gc, gitMerge(HgIdToGit[cs.Parents[1].ChangeSetId][1]))
				}
			}
		}
		lastGitId, err = hgOneCommitToGit(cs, func(err error) error {
			fmt.Println("hg2git: warning:", err.Error())
			return nil
		})
		if err != nil {
			return err
		}
		lastHgId = cs.ChangeSetId
		HgIdToGit[lastHgId] = [2]string{lastGitId, branchName}

		fmt.Printf("*** ChangeSetID: %s -> branch:%s commit:%s ***\n",
			lastHgId, branchName, lastGitId)

		for _, gc1 := range gc {
			gc1()
		}
		gc = gc[:0]
	}
	if branchName != "master" {
		com.Run("git", "branch", "-m", "master", "fork0000")
		com.Run("git", "branch", "-m", branchName, "master")
	}
	com.Run("git", "gc")
	return nil
}
