package scaffold

import (
	"io/fs"

	"github.com/rossedman/canon/canon"
)

// canonFiles reads every file under dir in the embedded canon.
func canonFiles(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := fs.WalkDir(canon.FS, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(canon.FS, p)
		out[p] = b
		return err
	})
	return out, err
}
