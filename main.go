package main

import (
	"github.com/urfave/cli"
	"log"
	"os"
	"strings"
)

type fileInfo struct {
	Name string
	Hash string
}

type dirInfo struct {
	Name  string
	Dirs  []dirInfo  `json:",omitempty"`
	Files []fileInfo `json:",omitempty"`
}

func main() {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:      "scan",
			ArgsUsage: "directory",
			Usage:     "scan all non-hidden files under one directory",
			Aliases:   []string{"s"},
			Action:    scan,
		},
		{
			Name:      "dedup",
			ArgsUsage: "dir1 [dir2 ...] ",
			Usage:     "search in given directories, show and delete duplicated one",
			Aliases:   []string{"c"},
			Action:    dedup,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "delete, d",
					Usage: "delete duplicated entries",
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func isPhotos(ext string) bool {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "jpg":
		fallthrough
	case "jpeg":
		fallthrough
	case "png":
		fallthrough
	case "heif":
		fallthrough
	case "heic":
		fallthrough
	case "webp":
		fallthrough
	case "dng":
		fallthrough
	case "bmp":
		fallthrough
	case "tif":
		fallthrough
	case "tiff":
		fallthrough
	case "enc":
		fallthrough
	case "arw":
		fallthrough
	case "gif":
		fallthrough
	case "webm":
		fallthrough
	case "mkv":
		fallthrough
	case "mp4":
		fallthrough
	case "mov":
		fallthrough
	case "mpg":
		fallthrough
	case "mpeg":
		fallthrough
	case "avi":
		fallthrough
	case "3gp":
		return true
	}
	return false
}

func mkFilename(fullname string) string {
	fullname = strings.TrimPrefix(fullname, string(os.PathSeparator))
	return strings.ReplaceAll(fullname, string(os.PathSeparator), "_") + ".json"
}
