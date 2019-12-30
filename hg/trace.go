package hg

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func run(name string, args ...string) error {
	cmd1 := exec.Command(name, args...)
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	cmd1.Stdin = os.Stdin

	fmt.Println()
	fmt.Print(name)
	for _, s := range args {
		fmt.Print(" ")
		if strings.IndexByte(s, ' ') >= 0 {
			fmt.Print("\"", s, "\"")
		} else {
			fmt.Print(s)
		}
	}
	fmt.Println()
	return cmd1.Run()
}

func quote(name string, args ...string) (string, error) {
	cmd1 := exec.Command(name, args...)
	cmd1.Stderr = os.Stderr
	cmd1.Stdin = os.Stdin
	output, err := cmd1.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

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
	return run("hg", "clone", src, dst)
}

func gitInit() error {
	if err := run("git", "init"); err != nil {
		return err
	}
	return run("git", "config", "--local", "core.autocrlf", "false")
}

func hgUpdateC(id string) error {
	return run("hg", "update", "-C", id)
}

func hgAdd(files ...string) error {
	args := []string{"add"}
	cont := ""
	for _, s := range files {
		fname := cont + s
		if _, err := os.Stat(fname); err != nil {
			cont = fname + " "
		} else {
			args = append(args, fname)
			cont = ""
		}
	}
	return run("git", args...)
}

func gitCommit(desc string, date time.Time, user string) (string, error) {
	err := run("git", "commit",
		"-m", desc,
		"--date", date.Format("Mon Jan 02 15:04:05 2006 -0700"),
		"--author", author(user))
	if err != nil {
		return "", err
	}
	return quote("git", "log", "-n", "1", "--format=%H")
}

func hgOneCommitToGit(cs *ChangeSet) (string, error) {
	if err := hgUpdateC(cs.ChangeSetId); err != nil {
		return "", err
	}
	if err := hgAdd(cs.Files...); err != nil {
		return "", err
	}
	return gitCommit(cs.Description, cs.Date, cs.User)
}

func gitMerge(id string) {
	run("git", "merge", "--no-commit", "--no-edit", id)
}

func gitCheckout(id, newbranch string) {
	run("git", "checkout", "-f", id)
	run("git", "checkout", "-b", newbranch)
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

	if err := gitInit(); err != nil {
		return err
	}

	lastHgId := rep.BySerial[0].ChangeSetId
	lastGitId, err := hgOneCommitToGit(rep.BySerial[0])
	if err != nil {
		return err
	}
	branchSerial := 0
	branchName := "master"

	HgIdToGit := map[string][2]string{
		lastHgId: [2]string{lastGitId, branchName}}

	for serial := 1; ; serial++ {
		cs, ok := rep.BySerial[serial]
		if !ok {
			break
		}
		if len(cs.Parents) >= 1 {
			if cs.Parents[0].ChangeSetId == lastHgId {
				if len(cs.Parents) >= 2 {
					gitMerge(HgIdToGit[cs.Parents[1].ChangeSetId][1])
				}
			} else if len(cs.Parents) >= 2 && cs.Parents[1].ChangeSetId == lastHgId {
				gitMerge(HgIdToGit[cs.Parents[0].ChangeSetId][1])
			} else {
				// new branch
				branchSerial++
				branchName = fmt.Sprintf("fork%d", branchSerial)
				gitCheckout(HgIdToGit[cs.Parents[0].ChangeSetId][0], branchName)
				if len(cs.Parents) >= 2 {
					gitMerge(HgIdToGit[cs.Parents[1].ChangeSetId][1])
				}
			}
		}
		lastGitId, err = hgOneCommitToGit(cs)
		if err != nil {
			return err
		}
		lastHgId = cs.ChangeSetId
		HgIdToGit[lastHgId] = [2]string{lastGitId, branchName}
	}

	return nil
}
