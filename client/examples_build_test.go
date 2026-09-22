package client_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExamplesBuild(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	tmp := t.TempDir()

	for _, name := range []string{"orderbook", "margin", "market"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("go", "build", "-o", filepath.Join(tmp, name), "./examples/"+name)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go build ./examples/%s: %v\n%s", name, err, out)
			}
		})
	}
}
