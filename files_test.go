package ember

import (
	"embed"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed example/dist
var example embed.FS

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.MkdirAll(dir+"/app/assets", 0755)
	assert.NoError(t, err)
	err = os.WriteFile(dir+"/app/index.html", []byte(indexHTML), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(dir+"/app/assets/script.js", []byte(scriptJS), 0644)
	assert.NoError(t, err)

	files := MustFiles(os.DirFS(dir), "app")
	assert.Equal(t, map[string]string{
		"index.html":       indexHTML,
		"assets/script.js": scriptJS,
	}, files)
}

func TestFilesExample(t *testing.T) {
	files := MustFiles(example, "example/dist")
	index, ok := files["index.html"]
	assert.True(t, ok)
	assert.NotEmpty(t, index)
}
