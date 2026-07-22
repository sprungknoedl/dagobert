package dissect

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupports(t *testing.T) {
	m := &Module{}

	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{"zip passes", model.Evidence{Name: "triage.zip"}, true},
		{"E01 passes", model.Evidence{Name: "disk.e01"}, true},
		{"vmdk passes", model.Evidence{Name: "disk.vmdk"}, true},
		{"ufdx passes", model.Evidence{Name: "case.ufdx"}, true},
		{"evtx rejected", model.Evidence{Name: "Security.evtx"}, false},
		{"no extension rejected", model.Evidence{Name: "README"}, false},
		{"non-evidence rejected", model.Indicator{Type: "Hash"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Supports(tc.obj))
		})
	}
}

func TestMessageFor(t *testing.T) {
	cases := []struct {
		name string
		desc string
		rec  map[string]any
		want string
	}{
		{
			"mft std uses path and ts_type, not ts_description",
			"filesystem/ntfs/mft/std",
			map[string]any{"path": `c:\$MFT`, "ts_type": "B", "ts_description": "ts"},
			`c:\$MFT was created ($STANDARD_INFORMATION)`,
		},
		{
			"mft std metadata changed",
			"filesystem/ntfs/mft/std",
			map[string]any{"path": `c:\$MFT`, "ts_type": "C"},
			`c:\$MFT was metadata changed ($STANDARD_INFORMATION)`,
		},
		{
			"mft filename uses path and ts_type",
			"filesystem/ntfs/mft/filename",
			map[string]any{"path": `c:\$MFT`, "ts_type": "M"},
			`c:\$MFT was modified ($FILE_NAME)`,
		},
		{
			"evtx uses EventID, Provider_Name and Channel",
			"filesystem/windows/evtx",
			map[string]any{"EventID": float64(1000), "Provider_Name": "Microsoft-Windows-Kernel", "Channel": "System"},
			"Event 1000 logged by Microsoft-Windows-Kernel (System)",
		},
		{
			"runkeys uses name and command executable+args",
			"windows/registry/run",
			map[string]any{
				"name":    "Steam",
				"command": map[string]any{"executable": `C:\Program Files (x86)\Steam\steam.exe`, "args": []any{"-silent"}},
			},
			`Autorun entry "Steam" launches C:\Program Files (x86)\Steam\steam.exe -silent (registry key last modified)`,
		},
		{
			"runkeys with no args",
			"windows/registry/run",
			map[string]any{
				"name":    "SecurityHealth",
				"command": map[string]any{"executable": `%windir%\system32\SecurityHealthSystray.exe`, "args": []any{}},
			},
			`Autorun entry "SecurityHealth" launches %windir%\system32\SecurityHealthSystray.exe (registry key last modified)`,
		},
		{
			"runkeys with nil command",
			"windows/registry/run",
			map[string]any{"name": "Empty", "command": nil},
			`Autorun entry "Empty" launches (no command) (registry key last modified)`,
		},
		{
			"service uses name and displayname",
			"windows/service",
			map[string]any{"name": "wmansvc", "displayname": "Windows Management Service"},
			`Service "wmansvc" (Windows Management Service) registry entry last modified`,
		},
		{
			"prefetch uses filename and runcount",
			"filesystem/ntfs/prefetch",
			map[string]any{"filename": `C:\Windows\explorer.exe`, "runcount": float64(42)},
			`C:\Windows\explorer.exe was executed (42 total runs recorded)`,
		},
		{
			"shimcache uses path",
			"windows/shimcache",
			map[string]any{"path": `C:\Program Files\app.exe`, "index": float64(3)},
			`C:\Program Files\app.exe referenced in ShimCache (file last modified; presence alone does not confirm execution)`,
		},
		{
			"userassist uses path and number_of_executions",
			"windows/registry/userassist",
			map[string]any{"path": `C:\Program Files\app.exe`, "number_of_executions": float64(5)},
			`C:\Program Files\app.exe was last executed (5 times total)`,
		},
		{
			"lnk file mtime",
			"windows/filesystem/lnk",
			map[string]any{"lnk_path": `C:\Users\tom\Recent\file.lnk`, "ts_description": "lnk_mtime"},
			`Shortcut C:\Users\tom\Recent\file.lnk was modified`,
		},
		{
			"lnk target ctime maps to the same short wording as lnk_ctime",
			"windows/filesystem/lnk",
			map[string]any{"lnk_path": `C:\Users\tom\Recent\file.lnk`, "ts_description": "target_ctime"},
			`Shortcut C:\Users\tom\Recent\file.lnk was created`,
		},
		{
			"unknown descriptor falls back to flattened fields",
			"windows/registry/unknown",
			map[string]any{"name": "Steam", "hostname": "MACWIN", "ts": "2024-01-01T00:00:00Z", "_type": "record"},
			"hostname=MACWIN, name=Steam",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, messageFor(tc.desc, tc.rec))
		})
	}
}

func TestRewrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "raw.jsonl")
	dst := filepath.Join(dir, "dst.jsonl")

	lines := []string{
		`{"_type": "recorddescriptor", "_data": ["filesystem/ntfs/mft/std", []]}`,
		`{"ts": "2022-11-11T18:58:52.437069+00:00", "ts_description": "ts", "ts_type": "B", "path": "c:\\$MFT", "_type": "record", "_recorddescriptor": ["filesystem/ntfs/mft/std", 3050922189]}`,
		`{"_type": "recorddescriptor", "_data": ["windows/registry/unknown", []]}`,
		`{"ts": "2023-02-03T07:36:28.873838+00:00", "ts_description": "ts", "name": "SecurityHealth", "_type": "record", "_recorddescriptor": ["windows/registry/unknown", 3137681893]}`,
	}
	require.NoError(t, os.WriteFile(src, []byte(join(lines)), 0o644))

	require.NoError(t, rewrite(src, dst))

	f, err := os.Open(dst)
	require.NoError(t, err)
	defer f.Close()

	var records []map[string]any
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &rec))
		records = append(records, rec)
	}
	require.NoError(t, scanner.Err())

	// recorddescriptor lines are skipped entirely
	require.Len(t, records, 2)

	assert.Equal(t, "2022-11-11T18:58:52.437069+00:00", records[0]["datetime"])
	assert.Equal(t, "ts", records[0]["timestamp_desc"])
	assert.Equal(t, `c:\$MFT was created ($STANDARD_INFORMATION)`, records[0]["message"])
	// original fields are preserved alongside the rewritten ones
	assert.Equal(t, `c:\$MFT`, records[0]["path"])
	// ts/ts_description are renamed, not duplicated
	assert.NotContains(t, records[0], "ts")
	assert.NotContains(t, records[0], "ts_description")

	assert.Equal(t, "2023-02-03T07:36:28.873838+00:00", records[1]["datetime"])
	assert.Equal(t, "ts", records[1]["timestamp_desc"])
	// untemplated descriptor falls back to flattened fields
	assert.Equal(t, "name=SecurityHealth", records[1]["message"])
	assert.NotContains(t, records[1], "ts")
	assert.NotContains(t, records[1], "ts_description")
}

func join(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}
