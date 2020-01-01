package main

import (
	"fmt"
	"os"
	"testing"
)

func TestLoadRepository(t *testing.T) {
	var rep Repository
	err := rep.Load("./testtarget",
		func(err error) error {
			fmt.Fprintf(os.Stderr, "ignore: [%s]\n", err)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	rep.Each(func(cs *ChangeSet) error {
		println("No=", cs.Serial)
		println("ChangeSetId=", cs.ChangeSetId)
		println("Date=", cs.Date.Format("2006/01/02 15:04:05 -0700 Mon"))
		println(cs.Description)
		println("------------")
		return nil
	})
}
