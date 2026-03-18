package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// JSONEntry is the JSON-serializable representation of a scanner Entry.
type JSONEntry struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	FileCount int       `json:"file_count"`
	IsDir     bool      `json:"is_dir"`
	Score     int       `json:"score"`
}

// WriteList writes entries in tab-delimited format (name\tpath\n) to w.
func WriteList(w io.Writer, entries []Entry) error {
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", e.Name, e.Path); err != nil {
			return err
		}
	}
	return nil
}

// WriteJSON writes entries as a JSON array to w.
func WriteJSON(w io.Writer, entries []Entry) error {
	out := make([]JSONEntry, len(entries))
	for i, e := range entries {
		out[i] = JSONEntry{
			Name:      e.Name,
			Path:      e.Path,
			Size:      e.Size,
			ModTime:   e.ModTime,
			FileCount: e.FileCount,
			IsDir:     e.IsDir,
			Score:     Score(e),
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
