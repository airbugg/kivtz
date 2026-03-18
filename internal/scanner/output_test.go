package scanner

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteList_formatsNameTabPath(t *testing.T) {
	entries := []Entry{
		{Name: "fish", Path: "/home/user/.config/fish", ModTime: time.Now()},
		{Name: ".gitconfig", Path: "/home/user/.gitconfig", ModTime: time.Now()},
	}

	var buf bytes.Buffer
	err := WriteList(&buf, entries)

	require.NoError(t, err)
	assert.Equal(t, "fish\t/home/user/.config/fish\n.gitconfig\t/home/user/.gitconfig\n", buf.String())
}

func TestWriteJSON_outputsParsableJSON(t *testing.T) {
	entries := []Entry{
		{Name: "fish", Path: "/home/user/.config/fish", Size: 1024, FileCount: 3, IsDir: true, ModTime: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, entries)

	require.NoError(t, err)

	// Parse back and verify
	var parsed []JSONEntry
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "fish", parsed[0].Name)
	assert.Equal(t, "/home/user/.config/fish", parsed[0].Path)
	assert.Equal(t, int64(1024), parsed[0].Size)
	assert.Equal(t, 3, parsed[0].FileCount)
	assert.True(t, parsed[0].IsDir)
}

func TestWriteList_emptyEntries(t *testing.T) {
	var buf bytes.Buffer
	err := WriteList(&buf, nil)

	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteJSON_emptyEntriesOutputsEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, nil)

	require.NoError(t, err)

	var parsed []JSONEntry
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Empty(t, parsed)
}

func TestWriteJSON_includesScore(t *testing.T) {
	entries := []Entry{
		{Name: "fish", Path: "/home/user/.config/fish", Size: 512, FileCount: 2, IsDir: true, ModTime: time.Now()},
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, entries)

	require.NoError(t, err)

	var parsed []JSONEntry
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Greater(t, parsed[0].Score, 0)
}
