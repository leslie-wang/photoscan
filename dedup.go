package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/manifoldco/promptui"
	sixel "github.com/mattn/go-sixel"
	"github.com/urfave/cli"
)

var tmpConvertFilename = os.TempDir() + string(os.PathSeparator) + "photoscan_dedup.jpg"

func dedup(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) == 0 {
		return errors.New("1 directory is needed at least")
	}

	fhash := map[string][]string{}
	for _, dir := range args {
		dinfo, err := parseFileHash(dir)
		if err != nil {
			return err
		}
		detectDup(fhash, dinfo)
	}

	for _, names := range fhash {
		if len(names) <= 1 {
			continue
		}
		err := promptDedup(names, ctx.Bool("delete"))
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

func promptDedup(names []string, delete bool) error {
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
		fmt.Printf("%s:\n", n)
		// use imagemagick to do conversion
		err = exec.Command("convert", n, "-resize", "100x100", tmpConvertFilename).Run()
		if err != nil {
			log.Printf("ERROR decode and render: %v\n", err)
			continue
		}
		img, err := parseImage(tmpConvertFilename)
		if err != nil {
			log.Printf("ERROR parse converted image: %v", err)
		}

		err = renderImage(img)
		if err != nil {
			log.Printf("ERROR encode sixel: %v", err)
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
		if delete {
			err = os.Remove(names[idx-1])
			if err != nil {
				log.Printf("ERROR delete %s: %s", names[idx-1], err)
			}
		} else {
			fmt.Println(items[idx])
		}
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
