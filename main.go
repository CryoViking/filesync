package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/mgutz/ansi"
	"github.com/urfave/cli/v2"
)

var DEBUG bool = false

func main() {
	app := &cli.App{
		Name:      "filesync",
		Usage:     "Sync the current folder to a destination folder on a remote machine when file changes occur",
		UsageText: "filesync --ssh-address <host> <dest_folder>",
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
			if resolved, err := filepath.EvalSymlinks(root_dir); err == nil {
				root_dir = resolved
			}

			options := Options{
				SyncedFolderDest: destFolder,
				ServerAddress:    c.String("ssh-address"),
				RootPath:         root_dir,
			}
			// Git mode only from the top of a work tree: the file list is relative to it, and so is
			// the destination.
			if top := gitRoot(root_dir); top != "" {
				if resolved, err := filepath.EvalSymlinks(top); err == nil && resolved == root_dir {
					options.GitRoot = root_dir
					options.Manifest = manifestPath(root_dir)
				}
			}
			if options.GitRoot != "" {
				log.Printf("%sMode%s: git source files (tracked, plus untracked files git doesn't ignore)\n", ansi.Cyan, ansi.DefaultFG)
			} else {
				log.Printf("%sMode%s: not the top of a git work tree; using .gitignore as an rsync filter\n", ansi.Yellow, ansi.DefaultFG)
			}
			// We set the verbosity of the program here.
			// NOTE: Yes this is a global variable but I'm a smart programmer...
			// why would I be setting it else where.
			DEBUG = c.Bool("verbose")

			watch(&options, newSyncer(&options), c.Duration("quiet"))
			return nil
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
			&cli.DurationFlag{
				Name:  "quiet",
				Value: 300_000_000, // 300ms
				Usage: "How long files must stay unchanged before a sync starts, so one save is one sync.",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
