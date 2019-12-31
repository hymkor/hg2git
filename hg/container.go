package hg

import (
	"fmt"
	"strconv"
	"strings"
)

type Repository struct {
	BySerial map[int]*ChangeSet
	ByHash   map[string]*ChangeSet
	ByTag    map[string]*ChangeSet
	Head     *ChangeSet
}

const nullCommit = "000000000000"

func (rep *Repository) Load(dir string, warn func(error) error) error {
	sets, err := LoadChangeSets(dir, warn)
	if err != nil {
		return err
	}
	rep.BySerial = make(map[int]*ChangeSet, len(sets))
	rep.ByHash = make(map[string]*ChangeSet, len(sets))
	rep.ByTag = make(map[string]*ChangeSet)

	sets = append(sets, &ChangeSet{Serial: -1, ChangeSetId: nullCommit})
	max := -1
	for _, set := range sets {
		rep.BySerial[set.Serial] = set
		rep.ByHash[set.ChangeSetId] = set

		for _, tag1 := range set.Tags {
			rep.ByTag[tag1] = set
		}
		if set.Serial > max {
			max = set.Serial
		}
	}
	if max >= 0 {
		rep.Head = rep.BySerial[max]
	} else {
		rep.Head = nil
	}
	for _, set := range sets {
		for _, idStr := range set.parentIDs {
			p := strings.Split(strings.TrimSpace(idStr), ":")
			idNum, err := strconv.Atoi(p[0])
			if err != nil {
				return err
			}
			if p, ok := rep.BySerial[idNum]; ok {
				set.Parents = append(set.Parents, p)
			} else {
				return fmt.Errorf("%d:%s: mercurial serial number %d(%s) not found.", set.Serial, set.ChangeSetId, idNum, idStr)
			}
		}
	}
	return nil
}

func (rep *Repository) Each(f func(*ChangeSet) error) error {
	serial := 0
	for {
		cs, ok := rep.BySerial[serial]
		if !ok {
			return nil
		}
		f(cs)
		serial++
	}
}
