package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/manifoldco/promptui"
	sixel "github.com/mattn/go-sixel"
	"github.com/urfave/cli"
)

func dedup(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) == 0 {
		return errors.New("2 directories are needed")
	}

	fhash := map[string][]string{}
	for _, dir := range args {
		dinfo, err := parseFileHash(dir)
		if err != nil {
			return err
		}
		detectDup(fhash, dinfo)
	}

	for hash, names := range fhash {
		if len(names) <= 1 {
			continue
		}
		err := promptDedup(hash, names)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseFileHash(dir string) (*dirInfo, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(mkFilename(dir))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dinfo := &dirInfo{}
	return dinfo, json.NewDecoder(f).Decode(dinfo)
}

func detectDup(fhash map[string][]string, dinfo *dirInfo) {
	for _, fi := range dinfo.Files {
		names, ok := fhash[fi.Hash]
		if !ok {
			names = []string{fi.Name}
		} else {
			names = append(names, fi.Name)
		}
		fhash[fi.Hash] = names
	}

	for _, di := range dinfo.Dirs {
		detectDup(fhash, &di)
	}
}

func promptDedup(hash string, names []string) error {
	log.Printf("%d dup found: %v", len(names), names)
	items := []string{"Keep all"}
	for _, n := range names {
		_, err := os.Stat(n)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%s is not exist anymore\n", n)
				continue
			}
		}
		img, err := parseImage(n)
		if err != nil {
			log.Printf("ERROR PARSE: %v", err)
			continue
		}

		fmt.Printf("%s:\n", n)
		err = renderImage(img)
		if err != nil {
			log.Printf("ERROR ENCODE SIXEL: %v", err)
			continue
		}
		items = append(items, "Delete '"+n+"'")
	}
	for len(items) > 2 {
		prompt := promptui.Select{
			Label: "Please confirm to proceed",
			Items: items,
			Size:  len(items),
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return err
		}
		if idx == 0 {
			return nil
		}
		fmt.Println(names[idx-1])
		names = append(names[:idx-1], names[idx:]...)
		items = append(items[:idx], items[idx+1:]...)
	}
	return nil
}

func parseImage(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func renderImage(img image.Image) error {
	buf := bufio.NewWriter(os.Stdout)
	defer buf.Flush()

	enc := sixel.NewEncoder(buf)
	enc.Dither = true
	return enc.Encode(img)
}
