package doct

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers -----------------------------------------------------------------

func buildZip(t *testing.T, files map[string]string) (io.ReaderAt, int64) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = io.WriteString(w, content)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	b := buf.Bytes()
	return bytes.NewReader(b), int64(len(b))
}

func readFromZip(t *testing.T, data []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		require.NoError(t, err)
		defer rc.Close()
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		return string(b)
	}
	t.Fatalf("file %q not found in zip", name)
	return ""
}

func writeTempOffice(t *testing.T, ext, contentFile, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test"+ext)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create(contentFile)
	require.NoError(t, err)
	_, err = io.WriteString(w, content)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return path
}

func writeTempDocx(t *testing.T, content string) string {
	return writeTempOffice(t, ".docx", "word/document.xml", content)
}

func writeTempOdt(t *testing.T, content string) string {
	return writeTempOffice(t, ".odt", "content.xml", content)
}

// --- xmlEscape ---------------------------------------------------------------

func TestXmlEscape(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"passthrough", "hello", "hello"},
		{"less-than", "<tag>", "&lt;tag&gt;"},
		{"ampersand", "a & b", "a &amp; b"},
		{"double-quote", `"quoted"`, "&#34;quoted&#34;"},
		{"combined", `<a href="x">`, "&lt;a href=&#34;x&#34;&gt;"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := xmlEscape(c.in)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// --- processZip --------------------------------------------------------------

func TestProcessZip(t *testing.T) {
	t.Run("copies all files", func(t *testing.T) {
		src, size := buildZip(t, map[string]string{
			"a.xml": "content a",
			"b.xml": "content b",
		})

		var dst bytes.Buffer
		err := processZip(src, size, &dst, func(_ *zip.FileHeader, r io.Reader, w io.Writer) error {
			_, err := io.Copy(w, r)
			return err
		})
		require.NoError(t, err)

		assert.Equal(t, "content a", readFromZip(t, dst.Bytes(), "a.xml"))
		assert.Equal(t, "content b", readFromZip(t, dst.Bytes(), "b.xml"))
	})

	t.Run("processor can transform content", func(t *testing.T) {
		src, size := buildZip(t, map[string]string{
			"content.xml": "hello",
			"meta.xml":    "meta",
		})

		var dst bytes.Buffer
		err := processZip(src, size, &dst, func(hdr *zip.FileHeader, r io.Reader, w io.Writer) error {
			if hdr.Name == "content.xml" {
				_, err := io.WriteString(w, "world")
				return err
			}
			_, err := io.Copy(w, r)
			return err
		})
		require.NoError(t, err)

		assert.Equal(t, "world", readFromZip(t, dst.Bytes(), "content.xml"))
		assert.Equal(t, "meta", readFromZip(t, dst.Bytes(), "meta.xml"))
	})

	t.Run("output has deterministic timestamps", func(t *testing.T) {
		src, size := buildZip(t, map[string]string{"a.xml": "x", "b.xml": "y"})

		var dst bytes.Buffer
		err := processZip(src, size, &dst, func(_ *zip.FileHeader, r io.Reader, w io.Writer) error {
			_, err := io.Copy(w, r)
			return err
		})
		require.NoError(t, err)

		zr, err := zip.NewReader(bytes.NewReader(dst.Bytes()), int64(dst.Len()))
		require.NoError(t, err)
		for _, f := range zr.File {
			assert.Equal(t, int64(0), f.Modified.Unix(), "file %s timestamp", f.Name)
		}
	})

	t.Run("processor error propagates", func(t *testing.T) {
		src, size := buildZip(t, map[string]string{"a.xml": "x"})
		sentinel := errors.New("processor failed")

		var dst bytes.Buffer
		err := processZip(src, size, &dst, func(_ *zip.FileHeader, _ io.Reader, _ io.Writer) error {
			return sentinel
		})
		assert.ErrorIs(t, err, sentinel)
	})
}

// --- OfficeTpl (MS / docx) ---------------------------------------------------

func TestOfficeTpl_Ms_Metadata(t *testing.T) {
	path := writeTempDocx(t, `<w:document><w:body><w:p><w:r><w:t>hello</w:t></w:r></w:p></w:body></w:document>`)
	tpl, err := LoadMsTemplate(path)
	require.NoError(t, err)

	assert.Equal(t, "test.docx", tpl.Name())
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", tpl.Type())
	assert.Equal(t, ".docx", tpl.Ext())
}

func TestOfficeTpl_Ms_Render(t *testing.T) {
	t.Run("substitutes template data", func(t *testing.T) {
		path := writeTempDocx(t, `<w:document><w:body><w:p><w:r><w:t>{{.Name}}</w:t></w:r></w:p></w:body></w:document>`)
		tpl, err := LoadMsTemplate(path)
		require.NoError(t, err)

		var out bytes.Buffer
		require.NoError(t, tpl.Render(&out, map[string]any{"Name": "Acme Inc"}, nil))

		assert.Contains(t, readFromZip(t, out.Bytes(), "word/document.xml"), "Acme Inc")
	})

	t.Run("xml funcmap escapes special characters", func(t *testing.T) {
		// The walker adds | xml automatically; verify it actually escapes.
		path := writeTempDocx(t, `<w:document>{{.Name}}</w:document>`)
		tpl, err := LoadMsTemplate(path)
		require.NoError(t, err)

		var out bytes.Buffer
		require.NoError(t, tpl.Render(&out, map[string]any{"Name": "<Acme & Co>"}, nil))

		assert.Contains(t, readFromZip(t, out.Bytes(), "word/document.xml"), "&lt;Acme &amp; Co&gt;")
	})

	t.Run("non-content files are copied unchanged", func(t *testing.T) {
		// Build a docx with an extra file alongside the content file
		path := filepath.Join(t.TempDir(), "multi.docx")
		f, err := os.Create(path)
		require.NoError(t, err)
		zw := zip.NewWriter(f)
		for name, body := range map[string]string{
			"word/document.xml": `{{.Name}}`,
			"word/styles.xml":   `<styles>static</styles>`,
		} {
			w, err := zw.Create(name)
			require.NoError(t, err)
			_, err = io.WriteString(w, body)
			require.NoError(t, err)
		}
		require.NoError(t, zw.Close())
		require.NoError(t, f.Close())

		tpl, err := LoadMsTemplate(path)
		require.NoError(t, err)

		var out bytes.Buffer
		require.NoError(t, tpl.Render(&out, map[string]any{"Name": "X"}, nil))

		assert.Equal(t, "<styles>static</styles>", readFromZip(t, out.Bytes(), "word/styles.xml"))
	})

	t.Run("missing key returns error", func(t *testing.T) {
		path := writeTempDocx(t, `<w:document>{{.Missing}}</w:document>`)
		tpl, err := LoadMsTemplate(path)
		require.NoError(t, err)

		var out bytes.Buffer
		assert.Error(t, tpl.Render(&out, map[string]any{}, nil))
	})
}

func TestLoadMsTemplate_Errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadMsTemplate("/nonexistent/path/file.docx")
		assert.Error(t, err)
	})

	t.Run("invalid Go template syntax", func(t *testing.T) {
		path := writeTempDocx(t, `<w:document>{{invalid }</w:document>`)
		_, err := LoadMsTemplate(path)
		assert.Error(t, err)
	})

	t.Run("marker spanning asymmetric structures", func(t *testing.T) {
		// A marker accidentally reaching into a hyperlink swallows tags whose
		// removal unbalances the tree; the well-formedness check must reject
		// the template at load time instead of producing corrupt documents.
		path := writeTempDocx(t, `<w:document><w:p>`+
			`<w:r><w:t>{{ .A</w:t></w:r>`+
			`<w:hyperlink><w:r><w:t>b }}</w:t></w:r></w:hyperlink>`+
			`</w:p></w:document>`)
		_, err := LoadMsTemplate(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid XML")
	})

	t.Run("legacy tr marker gets migration hint", func(t *testing.T) {
		path := writeTempDocx(t, `<w:document><w:p><w:t>{{tr range .X }}</w:t></w:p></w:document>`)
		_, err := LoadMsTemplate(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "legacy")
	})
}

// --- end to end: dedicated marker rows ----------------------------------------

func TestOfficeTpl_DedicatedRowLoop(t *testing.T) {
	path := writeTempDocx(t, `<w:document><w:body><w:tbl>`+
		`<w:tr><w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc></w:tr>`+
		`<w:tr><w:tc><w:p><w:r><w:t>{{ range .Assets }}</w:t></w:r></w:p></w:tc></w:tr>`+
		`<w:tr><w:tc><w:p><w:r><w:t>{{ .Name }}</w:t></w:r></w:p></w:tc></w:tr>`+
		`<w:tr><w:tc><w:p><w:r><w:t>{{ end }}</w:t></w:r></w:p></w:tc></w:tr>`+
		`</w:tbl></w:body></w:document>`)

	tpl, err := LoadMsTemplate(path)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, tpl.Render(&out, map[string]any{
		"Assets": []map[string]string{{"Name": "a1"}, {"Name": "a2"}, {"Name": "a3"}},
	}, nil))

	content := readFromZip(t, out.Bytes(), "word/document.xml")
	assert.NoError(t, validateXML([]byte(content)), "rendered XML must be well-formed")
	// one header row + one row per asset, no leftover marker rows
	assert.Equal(t, 4, strings.Count(content, "<w:tr>"))
	for _, name := range []string{"a1", "a2", "a3"} {
		assert.Contains(t, content, name)
	}
	assert.NotContains(t, content, "{{")
}

// --- shipped templates ---------------------------------------------------------

func TestLoadShippedTemplates(t *testing.T) {
	cases := map[string]func(string) (Template, error){
		"../../templates/Demo Word Report.docx":  LoadMsTemplate,
		"../../templates/Demo Writer Report.odt": LoadLibreTemplate,
		"../../templates/Demo Calc Report.ods":   LoadLibreTemplate,
	}
	for path, loadFn := range cases {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if _, err := os.Stat(path); err != nil {
				t.Skipf("template not present: %v", err)
			}
			_, err := loadFn(path)
			assert.NoError(t, err)
		})
	}
}

// --- OfficeTpl (Libre / odt) -------------------------------------------------

func TestOfficeTpl_Libre_Metadata(t *testing.T) {
	path := writeTempOdt(t, `<office:document-content><text:p>hello</text:p></office:document-content>`)
	tpl, err := LoadLibreTemplate(path)
	require.NoError(t, err)

	assert.Equal(t, "test.odt", tpl.Name())
	assert.Equal(t, "application/vnd.oasis.opendocument.text", tpl.Type())
	assert.Equal(t, ".odt", tpl.Ext())
}

func TestOfficeTpl_Libre_Render(t *testing.T) {
	t.Run("substitutes template data", func(t *testing.T) {
		path := writeTempOdt(t, `<office:document-content><text:p>{{.Name}}</text:p></office:document-content>`)
		tpl, err := LoadLibreTemplate(path)
		require.NoError(t, err)

		var out bytes.Buffer
		require.NoError(t, tpl.Render(&out, map[string]any{"Name": "Acme Inc"}, nil))

		assert.Contains(t, readFromZip(t, out.Bytes(), "content.xml"), "Acme Inc")
	})

	t.Run("missing key returns error", func(t *testing.T) {
		path := writeTempOdt(t, `<office:document-content>{{.Missing}}</office:document-content>`)
		tpl, err := LoadLibreTemplate(path)
		require.NoError(t, err)

		var out bytes.Buffer
		assert.Error(t, tpl.Render(&out, map[string]any{}, nil))
	})
}

func TestLoadLibreTemplate_Errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadLibreTemplate("/nonexistent/path/file.odt")
		assert.Error(t, err)
	})

	t.Run("invalid Go template syntax", func(t *testing.T) {
		path := writeTempOdt(t, `<office:document-content>{{invalid }</office:document-content>`)
		_, err := LoadLibreTemplate(path)
		assert.Error(t, err)
	})
}
