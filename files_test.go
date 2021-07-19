package ember

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.MkdirAll(dir+"/app/assets", 0755)
	assert.NoError(t, err)
	err = ioutil.WriteFile(dir+"/app/index.html", []byte(indexHTML), 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile(dir+"/app/assets/script.js", []byte(scriptJS), 0644)
	assert.NoError(t, err)

	files := MustFiles(os.DirFS(dir), "app")
	assert.Equal(t, map[string]string{
		"index.html":       indexHTML,
		"assets/script.js": scriptJS,
	}, files)
}
