package hg

type Repository struct {
	BySerial map[int]*ChangeSet
	ByHash   map[string]*ChangeSet
	Head     *ChangeSet
}

func (rep *Repository) Load(dir string, warn func(error) error) error {
	sets, err := LoadChangeSets(dir, warn)
	if err != nil {
		return err
	}
	rep.BySerial = make(map[int]*ChangeSet, len(sets))
	rep.ByHash = make(map[string]*ChangeSet, len(sets))
	max := -1
	for _, set := range sets {
		rep.BySerial[set.Serial] = set
		rep.ByHash[set.ChangeSetId] = set
		if set.Serial > max {
			max = set.Serial
		}
	}
	if max >= 0 {
		rep.Head = rep.BySerial[max]
	} else {
		rep.Head = nil
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
