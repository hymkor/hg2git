package hg

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

func trace1(cs *ChangeSet) error {
	stack := []*ChangeSet{}
	for cs.Parents != nil && len(cs.Parents) >= 1 {
		if len(cs.Parents) >= 2 {
			return fmt.Errorf("%s: not support branch", cs)
		} else {
			stack = append(stack, cs)
			cs = cs.Parents[0]
		}
	}
	if err := run("git", "init"); err != nil {
		return err
	}
	if err := run("git", "config", "--local", "core.autocrlf", "false"); err != nil {
		return err
	}
	for {
		if err := run("hg", "update", "-C", cs.ChangeSetId); err != nil {
			return err
		}
		args := []string{"add"}
		args = append(args, cs.Files...)
		if err := run("git", args...); err != nil {
			return err
		}
		err := run("git", "commit",
			"-m", cs.Description,
			"--date", cs.Date.Format("Mon Jan 02 15:04:05 2006 -0700"),
			"--author", author(cs.User))
		if err != nil {
			return err
		}
		commitid, err := quote("git", "log", "-n", "1", "--format=%H")
		if err == nil {
			fmt.Printf("git commit-id=[%s]\n", commitid)
		}
		if len(stack) <= 0 {
			return nil
		}
		cs = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}
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
	if err := run("hg", "clone", src, dst); err != nil {
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

	return trace1(rep.Head)
}
