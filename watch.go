package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watch syncs once at start, then again whenever files change. Changes are gathered until things
// have been quiet for `quiet`, so a save that writes a temp file, renames it and touches a backup
// is one sync, not three; and one sync runs at a time, with another after it if more changed.
func watch(o *Options, s *syncer, quiet time.Duration) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	if err := addTree(watcher, o.RootPath); err != nil {
		log.Fatal(err)
	}

	s.sync("start")

	var timer *time.Timer
	fire := make(chan struct{}, 1)
	pending := ""
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ignoredEvent(o, event.Name) {
				continue
			}
			// A new directory, perhaps with others already inside it (git checkout, mkdir -p):
			// watch the whole of it.
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := addTree(watcher, event.Name); err != nil {
						log.Println("Failed to watch directory:", event.Name, err)
					}
				}
			}
			if pending == "" {
				pending = relative(o, event.Name)
			} else if !strings.HasSuffix(pending, " and more") {
				pending += " and more"
			}
			if timer == nil {
				timer = time.AfterFunc(quiet, func() { fire <- struct{}{} })
			} else {
				timer.Reset(quiet)
			}
		case <-fire:
			reason := pending
			if reason == "" {
				reason = "changes"
			}
			pending = ""
			timer = nil
			s.sync(reason)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error:", err)
		}
	}
}

// addTree watches dir and every directory beneath it, except .git's insides.
func addTree(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // vanished while walking
		}
		if !info.IsDir() {
			return nil
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

// ignoredEvent drops editor droppings and anything inside .git. Changes in ignored paths still
// trigger a sync, which then copies nothing: cheaper than asking git about every event.
func ignoredEvent(o *Options, name string) bool {
	base := filepath.Base(name)
	if strings.HasSuffix(base, "~") || strings.HasSuffix(base, ".swp") || strings.HasSuffix(base, ".swx") || base == "4913" {
		return true
	}
	rel := relative(o, name)
	return rel == ".git" || strings.HasPrefix(rel, ".git/")
}

func relative(o *Options, name string) string {
	abs, err := filepath.Abs(name)
	if err != nil {
		return name
	}
	rel, err := filepath.Rel(o.RootPath, abs)
	if err != nil {
		return name
	}
	return rel
}
