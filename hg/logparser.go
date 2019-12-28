package hg

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ChangeSet struct {
	Serial      int
	ChangeSetId string
	User        string
	Date        time.Time
	Tags        []string
	Files       []string
	Description string

	Parents []*ChangeSet
}

type ErrColonNotFound string

func (e ErrColonNotFound) Error() string { return string(e) }

func parseChangeSetId(text string) (int, string, error) {
	text = strings.TrimSpace(text)
	ids := strings.SplitN(text, ":", 2)
	if len(ids) < 2 {
		return -1, text, ErrColonNotFound(text)
	}
	serial, err := strconv.Atoi(ids[0])
	if err != nil {
		return -1, text, fmt.Errorf("%s: %w", text, err)
	}
	return serial, ids[1], nil
}

func ReadChangeSets(r io.Reader, warn func(error) error) ([]*ChangeSet, error) {
	sc := bufio.NewScanner(r)
	changesets := make([]*ChangeSet, 0, 50)
	draft := new(ChangeSet)
	for sc.Scan() {
		line := sc.Text()
		field := strings.SplitN(line, ":", 2)
		if len(field) < 2 {
			if err := warn(ErrColonNotFound(line)); err != nil {
				return nil, err
			}
			continue
		}
		switch field[0] {
		case "changeset":
			var err error
			draft.Serial, draft.ChangeSetId, err = parseChangeSetId(field[1])
			if err != nil {
				if err = warn(err); err != nil {
					return nil, err
				}
				continue
			}
		case "tag":
			draft.Tags = strings.Fields(strings.TrimSpace(field[1]))
		case "user":
			draft.User = strings.TrimSpace(field[1])
		case "date":
			dateStr := strings.TrimSpace(field[1])
			var err error
			draft.Date, err = time.Parse("Mon Jan 02 15:04:05 2006 -0700", dateStr)
			if err != nil {
				if err = warn(err); err != nil {
					return nil, err
				}
			}

		case "files":
			draft.Files = strings.Fields(strings.TrimSpace(field[1]))
		case "description":
			var buffer strings.Builder
			buffer.Grow(256)
			for {
				if !sc.Scan() {
					draft.Description = strings.TrimSpace(buffer.String())
					goto exit
				}
				desc := sc.Text()
				if strings.HasPrefix(desc, "changeset:") {
					draft.Description = strings.TrimSpace(buffer.String())
					changesets = append(changesets, draft)

					draft = new(ChangeSet)
					var err error
					draft.Serial, draft.ChangeSetId, err = parseChangeSetId(desc[10:])
					if err != nil {
						if err = warn(err); err != nil {
							return nil, err
						}
						continue
					}
					break
				}
				if buffer.Len() > 0 {
					buffer.WriteByte('\n')
				}
				buffer.WriteString(desc)
			}
		default:
			if err := warn(fmt.Errorf("%s: not supported tag", field[0])); err != nil {
				return nil, err
			}
		}
	}
exit:
	changesets = append(changesets, draft)
	if sc.Err() != io.EOF {
		return changesets, sc.Err()
	}
	return changesets, nil
}

func LoadChangeSets(dir string, warn func(error) error) ([]*ChangeSet, error) {
	saveLang := os.Getenv("LANG")
	defer os.Setenv("LANG", saveLang)
	os.Setenv("LANG", "C")

	saveEncoding := os.Getenv("HGENCODING")
	defer os.Setenv("HGENCODING", saveEncoding)
	os.Setenv("HGENCODING", "utf-8")

	saveDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	defer os.Chdir(saveDir)
	os.Chdir(dir)

	cmd1 := exec.Command("hg", "log", "-v")
	cmd1.Stderr = os.Stderr
	cmd1.Stdin = os.Stdin
	hgLog, err := cmd1.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer hgLog.Close()

	if err := cmd1.Start(); err != nil {
		return nil, err
	}
	defer cmd1.Wait()

	return ReadChangeSets(hgLog, warn)
}
