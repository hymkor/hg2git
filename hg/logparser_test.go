package hg

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var sampleLog = `changeset:   36:09d459452118
tag:         tip
user:        HAYAMA_Kaoru <iyahaya@nifty.com>
date:        Wed Jan 04 14:26:31 2012 +0900
files:       readme_en.txt readme_ja.txt
description:
* Lua 5.2 向けにドキュメントを更新


changeset:   35:b89abc9d9f47
user:        HAYAMA_Kaoru <iyahaya@nifty.com>
date:        Wed Jan 04 11:09:32 2012 +0900
files:       luaone.c
description:
* bitand など、bit32ライブラリと重複する自前関数を削除


`

func testChangeSet(t *testing.T, changesets []*ChangeSet, err error) bool {
	if err != nil {
		t.Fatal(err.Error())
		return false
	}
	if changesets == nil {
		t.Fatal("changesets is nil")
		return false
	}
	if len(changesets) < 2 {
		t.Fatal("too few changesets")
		return false
	}
	for _, cs := range changesets {
		println("No=", cs.Serial)
		println("ChangeSetId=", cs.ChangeSetId)
		println("Date=", cs.Date.Format("2006/01/02 15:04:05 -0700 Mon"))
		println(cs.Description)
		println("------------")
	}
	return true
}

func TestReadChangeSets(t *testing.T) {
	changesets, err := ReadChangeSets(strings.NewReader(sampleLog),
		func(err error) error {
			fmt.Fprintf(os.Stderr, "ignore: [%s]\n", err.Error())
			return nil
		})
	testChangeSet(t, changesets, err)
}

func TestLoadChangeSets(t *testing.T) {
	changesets, err := LoadChangeSets("./testtarget",
		func(err error) error {
			fmt.Fprintf(os.Stderr, "ignore: [%s]\n", err)
			return nil
		})
	testChangeSet(t, changesets, err)
}
