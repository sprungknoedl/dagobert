package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprungknoedl/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type zipEntry struct {
	name     string
	content  []byte
	password string
	enc      zip.EncryptionMethod
}

func buildZip(t *testing.T, entries []zipEntry) *zip.Reader {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		var w io.Writer
		var err error
		if e.password != "" {
			w, err = zw.Encrypt(e.name, e.password, e.enc)
		} else {
			w, err = zw.Create(e.name)
		}
		require.NoError(t, err)
		_, err = w.Write(e.content)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return zr
}

func TestExtractEvtxNestedStructure(t *testing.T) {
	zr := buildZip(t, []zipEntry{
		{name: "hostA/System.evtx", content: []byte("system-a")},
		{name: "hostB/logs/Security.evtx", content: []byte("security-b")},
	})
	dst := t.TempDir()

	count, err := extractEvtx(zr, dst, "", 1<<20)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	a, err := os.ReadFile(filepath.Join(dst, "hostA", "System.evtx"))
	require.NoError(t, err)
	assert.Equal(t, "system-a", string(a))

	b, err := os.ReadFile(filepath.Join(dst, "hostB", "logs", "Security.evtx"))
	require.NoError(t, err)
	assert.Equal(t, "security-b", string(b))
}

func TestExtractEvtxFiltersNonEvtx(t *testing.T) {
	zr := buildZip(t, []zipEntry{
		{name: "keep.evtx", content: []byte("keep")},
		{name: "readme.txt", content: []byte("ignore me")},
	})
	dst := t.TempDir()

	count, err := extractEvtx(zr, dst, "", 1<<20)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = os.Stat(filepath.Join(dst, "keep.evtx"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dst, "readme.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestExtractEvtxNoMatches(t *testing.T) {
	zr := buildZip(t, []zipEntry{
		{name: "readme.txt", content: []byte("nothing here")},
	})
	dst := t.TempDir()

	count, err := extractEvtx(zr, dst, "", 1<<20)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	entries, err := os.ReadDir(dst)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestExtractEvtxRejectsPathTraversal(t *testing.T) {
	zr := buildZip(t, []zipEntry{
		{name: "../evil.evtx", content: []byte("escape")},
	})
	dst := t.TempDir()

	_, err := extractEvtx(zr, dst, "", 1<<20)
	require.Error(t, err)

	_, err = os.Stat(filepath.Join(filepath.Dir(dst), "evil.evtx"))
	assert.True(t, os.IsNotExist(err))
}

func TestExtractEvtxBudgetExceeded(t *testing.T) {
	content := bytes.Repeat([]byte("x"), 100)
	zr := buildZip(t, []zipEntry{
		{name: "big.evtx", content: content},
	})
	dst := t.TempDir()

	_, err := extractEvtx(zr, dst, "", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum decompressed size")

	data, readErr := os.ReadFile(filepath.Join(dst, "big.evtx"))
	if readErr == nil {
		assert.Less(t, len(data), len(content))
	}
}

func TestExtractEvtxPasswordCorrect(t *testing.T) {
	zr := buildZip(t, []zipEntry{
		{name: "secure.evtx", content: []byte("plaintext"), password: "s3cret", enc: zip.AES256Encryption},
	})
	dst := t.TempDir()

	count, err := extractEvtx(zr, dst, "s3cret", 1<<20)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	data, err := os.ReadFile(filepath.Join(dst, "secure.evtx"))
	require.NoError(t, err)
	assert.Equal(t, "plaintext", string(data))
}

func TestExtractEvtxPasswordWrongOrMissing(t *testing.T) {
	cases := []struct {
		name     string
		password string
		want     string
	}{
		{"wrong password", "nope", "wrong password"},
		{"missing password", "", "password protected"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			zr := buildZip(t, []zipEntry{
				{name: "secure.evtx", content: []byte("plaintext"), password: "s3cret", enc: zip.AES256Encryption},
			})
			dst := t.TempDir()

			_, err := extractEvtx(zr, dst, tc.password, 1<<20)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
