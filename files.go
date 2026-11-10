package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// The files to sync are the ones git considers source: tracked files plus untracked files that
// aren't ignored. Git is asked rather than its rules re-implemented, because rsync's filter syntax
// only looks like .gitignore: it has no negation (`!src/database/bin/**`), so a file git tracks
// under an ignored directory was never synced, and its --delete reached into build output on the
// remote that was never ours to touch.

// gitRoot returns the top of the git work tree containing dir, or "" if there isn't one.
func gitRoot(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// sourceFiles lists the files under root that git would call source, relative to root, sorted.
// A tracked file deleted from the work tree is left out: it is gone, and the diff against the last
// manifest deletes it remotely.
func sourceFiles(root string) ([]string, error) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var files []string
	for _, f := range splitNul(out) {
		if seen[f] {
			continue // a file with a merge conflict is listed once per stage
		}
		seen[f] = true
		if info, err := os.Lstat(filepath.Join(root, f)); err != nil || info.IsDir() {
			continue // deleted, or a submodule's directory
		}
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// removed returns the paths in previous that are no longer in current. Both must be sorted.
func removed(previous, current []string) []string {
	var gone []string
	i, j := 0, 0
	for i < len(previous) {
		switch {
		case j >= len(current) || previous[i] < current[j]:
			gone = append(gone, previous[i])
			i++
		case previous[i] == current[j]:
			i++
			j++
		default:
			j++
		}
	}
	return gone
}

// safeRelative reports whether a path from git is one we may hand to `rm` on the remote: relative,
// and never climbing out of the destination.
func safeRelative(p string) bool {
	if p == "" || filepath.IsAbs(p) {
		return false
	}
	clean := filepath.Clean(p)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func splitNul(b []byte) []string {
	var out []string
	for _, part := range bytes.Split(b, []byte{0}) {
		if len(part) > 0 {
			out = append(out, string(part))
		}
	}
	return out
}

func joinNul(paths []string) []byte {
	var buf bytes.Buffer
	for _, p := range paths {
		buf.WriteString(p)
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

// The manifest is what was last synced, kept inside .git so it is never synced itself and is per
// work tree. With it, a file deleted while filesync wasn't running is still deleted remotely at
// the next start.

func manifestPath(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--git-path", "filesync-manifest").Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p
}

func readManifest(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	files := splitNul(b)
	sort.Strings(files)
	return files
}

func writeManifest(path string, files []string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, joinNul(files), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
