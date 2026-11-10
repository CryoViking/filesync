package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/mgutz/ansi"
)

type Options struct {
	SyncedFolderDest string
	ServerAddress    string
	RootPath         string
	// GitRoot is RootPath's git work tree, when RootPath is its top. Empty means not a git
	// repository, and the old .gitignore-as-rsync-filter behaviour is used.
	GitRoot  string
	Manifest string
}

// syncer copies source files to the remote and deletes the ones that have gone. It is only ever
// used from one goroutine at a time.
type syncer struct {
	o        *Options
	previous []string
}

func newSyncer(o *Options) *syncer {
	s := &syncer{o: o}
	if o.Manifest != "" {
		s.previous = readManifest(o.Manifest)
	}
	return s
}

func (s *syncer) sync(reason string) {
	log.Printf("%sSyncing%s: %s to %s:%s\n", ansi.Cyan, ansi.DefaultFG, reason, s.o.ServerAddress, s.o.SyncedFolderDest)
	var err error
	if s.o.GitRoot == "" {
		err = s.syncWithIgnoreFilter()
	} else {
		err = s.syncSourceFiles()
	}
	if err != nil {
		log.Printf("%sERROR%s: %v", ansi.Red, ansi.DefaultFG, err)
		return
	}
	log.Printf("%sSynced%s: %s\n", ansi.Green, ansi.DefaultFG, reason)
}

func (s *syncer) syncSourceFiles() error {
	files, err := sourceFiles(s.o.GitRoot)
	if err != nil {
		return fmt.Errorf("listing source files: %w", err)
	}

	// --files-from with -0: only these files, so nothing else on the remote is ever written or
	// deleted. Build output there is left alone, whoever owns it.
	rsync := exec.Command("rsync", "-az", "-0", "--files-from=-", ".", fmt.Sprintf("%s:%s", s.o.ServerAddress, s.o.SyncedFolderDest))
	rsync.Dir = s.o.GitRoot
	rsync.Stdin = bytes.NewReader(joinNul(files))
	if out, err := rsync.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync: %v, output: %s", err, out)
	} else if DEBUG {
		log.Printf("%sRSYNC%s: %s", ansi.Cyan, ansi.DefaultFG, out)
	}

	if err := s.deleteRemote(removed(s.previous, files)); err != nil {
		return err
	}
	s.previous = files
	if s.o.Manifest != "" {
		if err := writeManifest(s.o.Manifest, files); err != nil {
			log.Printf("%sWARNING%s: could not save the manifest: %v", ansi.Yellow, ansi.DefaultFG, err)
		}
	}
	return nil
}

// deleteRemote removes files that were synced before and are gone now, then any directories
// that leaves empty. The macOS rsync (openrsync) has no --delete-missing-args, so this is done
// over plain ssh.
func (s *syncer) deleteRemote(gone []string) error {
	var paths []string
	for _, p := range gone {
		if safeRelative(p) {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	for _, p := range paths {
		log.Printf("%sDeleting%s: %s\n", ansi.Yellow, ansi.DefaultFG, p)
	}
	// The list arrives once, on stdin, NUL-separated: a branch switch can delete thousands of files,
	// too many for a command line. rmdir -p climbs from each emptied directory and stops at the
	// first that still holds something, so it never leaves the destination.
	script := fmt.Sprintf(`cd %s || exit 1
t=$(mktemp) && cat >"$t" || exit 1
xargs -0 rm -f -- <"$t"
tr '\0' '\n' <"$t" | xargs -d '\n' -n1 dirname | sort -u | xargs -r -d '\n' rmdir -p --ignore-fail-on-non-empty 2>/dev/null
rm -f "$t"
true`, shellQuote(s.o.SyncedFolderDest))
	ssh := exec.Command("ssh", s.o.ServerAddress, script)
	ssh.Stdin = bytes.NewReader(joinNul(paths))
	if out, err := ssh.CombinedOutput(); err != nil {
		return fmt.Errorf("deleting on the remote: %v, output: %s", err, out)
	}
	return nil
}

// syncWithIgnoreFilter is the original behaviour, for a folder that isn't a git repository.
func (s *syncer) syncWithIgnoreFilter() error {
	rsync := exec.Command("rsync", "-avz", ".", fmt.Sprintf("%s:%s", s.o.ServerAddress, s.o.SyncedFolderDest),
		"--include=**.gitignore", "--exclude=/.git", "--filter=:- .gitignore", "--delete-after")
	rsync.Dir = s.o.RootPath
	out, err := rsync.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync: %v, output: %s", err, out)
	}
	if DEBUG {
		log.Printf("%sRSYNC%s: %s", ansi.Cyan, ansi.DefaultFG, out)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
