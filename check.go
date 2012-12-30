// 29 december 2012
package main

import (
	"fmt"
	"os"
	"io"
	"path/filepath"
	"archive/zip"
	"strconv"
	"crypto/sha1"
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
		log.Fatalf("hex decode error reading sha1 (%q): %v", expectstring, err)
	}

	f, err := zf.Open()
	if err != nil {
		return false, fmt.Errorf("could not open given zip file entry: %v", err)
	}
	defer f.Close()

	sha1hash.Reset()
	n, err := io.Copy(sha1hash, f)
	if err != nil {
		return false, fmt.Errorf("could not read given zip file entry: %v", err)
	}
	// TODO could we have integer size/signedness conversion failure here? zf.UncompressedSize is not an int64
	if n != int64(zf.UncompressedSize) {
		return false, fmt.Errorf("short read from zip file or write to hash but no error returned (expected %d bytes; got %d)", int64(zf.UncompressedSize), n)
	}

	return bytes.Equal(expected, sha1hash.Sum(nil)), nil
}

func (g *Game) Filename(rompath string) string {
	return filepath.Join(rompath, g.Name + ".zip")
}

func (g *Game) CheckIn(rompath string) (bool, error) {
	zipname := g.Filename(rompath)
	f, err := zip.OpenReader(zipname)
	if os.IsNotExist(err) {		// if the file does not exist, try the next rompath
		return false, nil
	}
	if err != nil {			// something different happened
		return false, fmt.Errorf("could not open zip file %s: %v", zipname, err)
	}
	defer f.Close()

	// populate list of files
	var files = make(map[string]*zip.File)
	for _, file := range f.File {
		files[file.Name] = file
	}

	// now check
	for _, rom := range g.ROMs {
		file, ok := files[rom.Name]
		if !ok {				// not in archive
			return false, nil
		}
		if file.UncompressedSize != rom.Size {
			return false, nil
		}
		if !crc32match(file.CRC32, rom.CRC32) {
			return false, nil
		}
		good, err := sha1check(file, rom.SHA1)
		if err != nil {
			return false, fmt.Errorf("could not calculate SHA-1 sum of %s in %s: %v", g.Name, zipname, err)
		}
		if !good {
			return false, nil
		}
		delete(files, rom.Name)		// mark as done
	}

	// if we reached here everything we know about checked out, so if there are any leftover files in the zip, that means something is wrong
	return len(files) == 0, nil
}
