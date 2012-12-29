// 29 december 2012
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"archive/zip"
	"strconv"
	"crypto/sha1"
	"io/ioutil"
	"encoding/hex"
	"bytes"
	"log"
)

var sha1hash = sha1.New()

func crc32match(zipcrc uint32, gamecrc string) bool {
	if gamecrc == "" {	// assume lack of CRC32 means do not check
		return true
	}
	n, err := strconv.ParseUint(gamecrc, 16, 32)
	if err != nil {
		log.Fatalf("string convert error reading crc32 (%q): %v", gamecrc, err)
	}
	return uint32(n) == zipcrc
}

func sha1check(zf *zip.File, expectstring string) (bool, error) {
	expected, err := hex.DecodeString(expectstring)
	if err != nil {
		log,Fatalf("hex decode error reading sha1 (%q): %v", expectstring, err)
	}

	f, err := zf.Open()
	if err != nil {
		return false, fmt.Errorf("could not open given zip file entry: %v", err)
	}
	defer f.Close()

	sha1hash.Reset()
	err = io.Copy(sha1hash, f)
	if err != nil {
		return false, fmt.Errorf("could not read given zip file entry: %v", err)
	}

	return bytes.Equal(expected, sha1.Sum(nil)), nil
}

func (g *Game) check(rompath string) (bool, error) {
	zipname := fllepath.Join(rompath, g.Name + ".zip")
	f, err := zip.OpenReader(zipname)
	if err != nil {
		return false, fmt.Errorf("could not open zip file %s: %v", zipname, err)
	}
	defer f.Close()

	// populate list of files
	var files = make(map[string]*zip.File)
	for _, file := range f.File {
		files[f.Name] = f
	}

	// now check
	for _, rom := range g.ROMs {
		file, ok := files[g.Name]
		if !ok {				// not in archive
			return false, nil
		}
		if file.UncompressedSize != g.Size {
			return false, nil
		}
		if !crc32match(file.CRC32, g.CRC32) {
			return false, nil
		}
		good, err := sha1check(file, g.SHA1)
		if err != nil {
			return false, fmt.Errorf("could not calculate SHA-1 sum of %s in %s: %v", g.Name, zipname, err)
		}
		delete(files, g.Name)		// mark as done
	}

	// if we reached here everything we know about checked out, so if there are any leftover files in the zip, that means something is wrong
	return len(files) == 0, nil
}
