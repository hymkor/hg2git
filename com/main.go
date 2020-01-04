package com

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func dump(name string, args []string) {
	fmt.Printf("\n$ %s", name)
	for _, s := range args {
		fmt.Print(" ")
		if strings.IndexByte(s, ' ') >= 0 {
			fmt.Print("\"", s, "\"")
		} else {
			fmt.Print(s)
		}
	}
	fmt.Println()
}

func Run(name string, args ...string) error {
	cmd1 := exec.Command(name, args...)
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	cmd1.Stdin = os.Stdin

	dump(name, args)
	return cmd1.Run()
}

func Quote(name string, args ...string) (string, error) {
	cmd1 := exec.Command(name, args...)
	cmd1.Stderr = os.Stderr
	cmd1.Stdin = os.Stdin
	dump(name, args)
	output, err := cmd1.Output()
	if err != nil {
		return "", err
	}
	result := strings.TrimSpace(string(output))
	for _, line := range strings.Split(result, "\n") {
		fmt.Printf("> %s\n", line)
	}
	return result, nil
}
