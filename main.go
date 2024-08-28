package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mgutz/ansi"
	"github.com/urfave/cli/v2"
)

var DEBUG bool = false

type Options struct {
	SyncedFolderDest string
	ServerAddress    string
	RootPath         string
}

func trim_root_folder(o *Options, filepath string) string {
	remaining_path := strings.TrimPrefix(filepath, o.RootPath)

	// Ensure that it starts with a '/'
	if !strings.HasPrefix(remaining_path, "/") {
		remaining_path = "/" + remaining_path
	}
	return remaining_path
}

func sync_files(o *Options, filepath string) {
	// Construct the rsync command
	file_dest := o.SyncedFolderDest + trim_root_folder(o, filepath)
	rsync_cmd := exec.Command(
		"rsync",
		"-avz",
		".",
		fmt.Sprintf("%s:%s", o.ServerAddress, o.SyncedFolderDest),
		"--include=**.gitignore",
		"--exclude=/.git",
		"--filter=:- .gitignore",
		"--delete-after",
	)

	// Print the curated output.
	log.Printf("%sSyncing%s: %s to location: %s:%s\n",
		ansi.Cyan,
		ansi.DefaultFG,
		filepath,
		o.ServerAddress,
		file_dest,
	)

	// Run the command and capture any errors
	output, err := rsync_cmd.CombinedOutput()
	if err != nil {
		log.Printf("%sERROR%s: %v, output: %s",
			ansi.Red,
			ansi.DefaultFG,
			err,
			output,
		)
	} else {
		log.Printf("%sSynced%s: %s\n",
			ansi.Green,
			ansi.DefaultFG,
			filepath,
		)
		if DEBUG {
			log.Printf("%sRSYNC%s: %s",
				ansi.Cyan,
				ansi.DefaultFG,
				output,
			)
		}
	}
}

func file_watcher(o *Options) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	defer close(done)

	absPath, err := filepath.Abs(".")
	if err != nil {
		log.Fatal(err)
	}

	// Add all subdirectories to the watcher
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Watch directories only
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start a goroutine to watch for events
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				abs_name_path, err := filepath.Abs(event.Name)
				if err != nil {
					log.Println(err)
					continue
				}
				// Don't do anything if the file is a temp nvim file
				if abs_name_path[len(abs_name_path)-1:] == "~" {
					continue
				}
				// Sync on Create, Write, Rename, or Remove events
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Remove == fsnotify.Remove {
					sync_files(o, abs_name_path)
				}
				// If a new directory is created, add it to the watcher
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(abs_name_path)
					if err == nil && info.IsDir() {
						if err := watcher.Add(abs_name_path); err != nil {
							log.Println("Failed to add directory:", abs_name_path, err)
						}
					}
				}
			case err := <-watcher.Errors:
				log.Println("Error:", err)
			}
		}
	}()

	<-done
}

func main() {
	app := &cli.App{
		Name:      "filesync",
		Usage:     "Sync the current folder to a destination folder on a remote machine when file changes occur",
		UsageText: "filesync <dest_folder>",
		Action: func(c *cli.Context) error {
			destFolder := c.Args().First()
			if destFolder == "" {
				log.Fatalf("Error: Missing destination folder argument.\n" +
					"Please specify the destination folder as the first argument.\n" +
					"Use the -h flag to see available options.")
			}

			root_dir, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}

			options := Options{
				SyncedFolderDest: destFolder,
				ServerAddress:    c.String("ssh-address"),
				RootPath:         root_dir,
			}
			// We set the verbosity of the program here.
			// NOTE: Yes this is a global variable but I'm a smart programmer...
			// why would I be setting it else where.
			DEBUG = c.Bool("verbose")

			go file_watcher(&options)
			select {} // Keep the main function running
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "ssh-address",
				Value:    "localhost",
				Required: true, // Make the flag required
				Usage:    "The ssh address of the machine to sync files to. This can be a hostname given in ssh config.",
			},
			&cli.BoolFlag{
				Name:     "verbose",
				Value:    false,
				Required: false,
				Usage:    "Set the verbosity of the program, if enabled rsync output will also be printed.",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
