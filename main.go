package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
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

type dirSorter struct {
	dirInfos *dirInfo
	sortDir  bool
}

// Len is part of sort.Interface.
func (s *dirSorter) Len() int {
	if s.sortDir {
		return len(s.dirInfos.Dirs)
	}
	return len(s.dirInfos.Files)
}

// Swap is part of sort.Interface.
func (s *dirSorter) Swap(i, j int) {
	if s.sortDir {
		s.dirInfos.Dirs[i], s.dirInfos.Dirs[j] = s.dirInfos.Dirs[j], s.dirInfos.Dirs[i]
	} else {
		s.dirInfos.Files[i], s.dirInfos.Files[j] = s.dirInfos.Files[j], s.dirInfos.Files[i]
	}
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *dirSorter) Less(i, j int) bool {
	if s.sortDir {
		return s.dirInfos.Dirs[i].Name < s.dirInfos.Dirs[j].Name
	}
	return s.dirInfos.Files[i].Name < s.dirInfos.Files[j].Name
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
			Name:      "compare",
			ArgsUsage: "dir1 dir2 ",
			Usage:     "compare 2 directories, show and delete duplicated one",
			Aliases:   []string{"c"},
			Action:    scan,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func scan(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return errors.New("must input 1 directory name")
	}

	dirname, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	result, err := scanDir(args[0])
	if err != nil {
		return err
	}
	dirname = strings.TrimPrefix(dirname, string(os.PathSeparator))
	filename := strings.ReplaceAll(dirname, string(os.PathSeparator), "_")
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func scanDir(dirname string) (dirInfo, error) {
	dinfos := dirInfo{Name: dirname, Dirs: []dirInfo{}, Files: []fileInfo{}}
	finfoCh := make(chan fileInfo)
	errCh := make(chan error)
	count := 0

	entries, err := os.ReadDir(dirname)
	if err != nil {
		return dinfos, err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return dinfos, err
		}

		// skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			continue
		}

		fullname := filepath.Join(dirname, info.Name())
		if info.IsDir() {
			subInfo, err := scanDir(fullname)
			if err != nil {
				return dinfos, err
			}
			if len(subInfo.Dirs) != 0 || len(subInfo.Files) != 0 {
				dinfos.Dirs = append(dinfos.Dirs, subInfo)
			}
			continue
		}

		if !info.Mode().IsRegular() {
			continue
		}

		if !isPhotos(filepath.Ext(info.Name())) {
			continue
		}

		count++
		go func() {
			fi, err := calHash(fullname)
			if err != nil {
				errCh <- err
			} else {
				finfoCh <- fi
			}
		}()
	}

	for i := 0; i < count; i++ {
		select {
		case finfo := <-finfoCh:
			dinfos.Files = append(dinfos.Files, finfo)
		case err := <-errCh:
			return dinfos, err
		}
	}

	ds := &dirSorter{dirInfos: &dinfos}
	sort.Sort(ds)
	ds.sortDir = true
	sort.Sort(ds)

	return dinfos, nil
}

func calHash(filename string) (fi fileInfo, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	h := md5.New()
	if _, err = io.Copy(h, f); err != nil {
		return
	}

	fi.Name = filename
	fi.Hash = fmt.Sprintf("%x", h.Sum(nil))

	return
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
