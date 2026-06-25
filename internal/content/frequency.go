package content

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// loadFrequency reads a target-language word-frequency list from
// content/<lang>/frequency.tsv (rank<TAB>word<TAB>reading<TAB>count), skipping
// blank lines and '#' comments. The list is parsed and validated but is not yet
// wired into card scheduling — that is the frequency-backfill follow-up.
func loadFrequency(fsys fs.FS, lang string) ([]model.FreqEntry, error) {
	file := path.Join("content", lang, "frequency.tsv")
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}

	var entries []model.FreqEntry
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Split(text, "\t")
		if len(fields) != 4 {
			return nil, fmt.Errorf("%s:%d: expected 4 tab-separated fields, got %d", file, line, len(fields))
		}
		rank, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: invalid rank %q: %w", file, line, fields[0], err)
		}
		count, err := strconv.Atoi(fields[3])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: invalid count %q: %w", file, line, fields[3], err)
		}
		entries = append(entries, model.FreqEntry{
			Rank:    rank,
			Word:    fields[1],
			Reading: fields[2],
			Count:   count,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	return entries, nil
}
