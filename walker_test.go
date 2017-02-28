package diffcopy

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/containerd/fs"
	"github.com/stretchr/testify/assert"
)

func TestWalkerSimple(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
file foo2
`)

}

func TestWalkerInclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar dir",
		"ADD bar/foo file",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePaths: []string{"bar", "bar/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

}

func TestWalkerExclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo2 file",
		"ADD foo/bar2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		ExcludePatterns: []string{"foo*", "!foo/bar2"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
dir foo
file foo/bar2
`, string(b.Bytes()))

}

func bufWalk(buf *bytes.Buffer) filepath.WalkFunc {
	return func(path string, fi os.FileInfo, err error) error {
		t := "file"
		if fi.IsDir() {
			t = "dir"
		}
		fmt.Fprintf(buf, "%s %s\n", t, path)
		return nil
	}
}

func tmpDir(inp []*change) (dir string, retErr error) {
	tmpdir, err := ioutil.TempDir("", "diff")
	if err != nil {
		return "", err
	}
	defer func() {
		if retErr != nil {
			os.RemoveAll(tmpdir)
		}
	}()
	for _, c := range inp {
		if c.kind == fs.ChangeKindAdd {
			p := filepath.Join(tmpdir, c.path)
			if c.fi.IsDir() {
				if err := os.Mkdir(p, 0700); err != nil {
					return "", err
				}
			} else {
				if _, err := os.Create(p); err != nil {
					return "", err
				}
			}
		}
	}
	return tmpdir, nil
}