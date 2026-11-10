package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRemoved(t *testing.T) {
	cases := []struct {
		previous, current, want []string
	}{
		{nil, []string{"a"}, nil},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, nil},
		{[]string{"a", "b", "c"}, []string{"a", "c"}, []string{"b"}},
		{[]string{"a", "b"}, []string{"b", "c"}, []string{"a"}},
		{[]string{"a", "b"}, nil, []string{"a", "b"}},
		{[]string{"b", "d"}, []string{"a", "c", "e"}, []string{"b", "d"}},
	}
	for _, c := range cases {
		if got := removed(c.previous, c.current); !reflect.DeepEqual(got, c.want) {
			t.Errorf("removed(%v, %v) = %v, want %v", c.previous, c.current, got, c.want)
		}
	}
}

func TestSafeRelative(t *testing.T) {
	for p, want := range map[string]bool{
		"src/a.cs": true, "a": true, "": false, ".": false, "..": false,
		"../x": false, "/etc/passwd": false, "a/../../x": false, "a/../b": true,
	} {
		if got := safeRelative(p); got != want {
			t.Errorf("safeRelative(%q) = %v, want %v", p, got, want)
		}
	}
}

// The case that started this: a directory ignored by one pattern and brought back by a negation.
// rsync's filter syntax has no negation, so it never synced; git's list includes it.
func TestSourceFilesFollowsGitIncludingNegation(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	write := func(p, body string) {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	write(".gitignore", "[Bb]in/\npublish/\n!src/database/bin/\n!src/database/bin/**\n")
	write("src/app/Program.cs", "x")
	write("src/app/bin/Release/app.dll", "x")      // build output: ignored
	write("src/database/bin/muli-db-migrate", "x") // negated back in
	write("publish/out.deb", "x")                  // ignored
	write("notes.txt", "x")                        // untracked, not ignored
	write("gone.txt", "x")
	run("add", ".gitignore", "src/app/Program.cs", "src/database/bin/muli-db-migrate", "gone.txt")
	if err := os.Remove(filepath.Join(root, "gone.txt")); err != nil { // tracked, deleted from the work tree
		t.Fatal(err)
	}

	got, err := sourceFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".gitignore", "notes.txt", "src/app/Program.cs", "src/database/bin/muli-db-migrate"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sourceFiles = %v, want %v", got, want)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest")
	files := []string{"a b.txt", "src/c.cs", "z"}
	if err := writeManifest(path, files); err != nil {
		t.Fatal(err)
	}
	if got := readManifest(path); !reflect.DeepEqual(got, files) {
		t.Errorf("readManifest = %v, want %v", got, files)
	}
	if got := readManifest(filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Errorf("a missing manifest should read as nil, got %v", got)
	}
}
