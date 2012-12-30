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
	"strings"
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

func (g *Game) checkOneZip(zipname string, roms map[string]*ROM, isParent bool) (bool, error) {
	f, err := zip.OpenReader(zipname)
	if os.IsNotExist(err) {		// if the file does not exist, try the next rompath
		return false, nil
	}
	if err != nil {			// something different happened
		return false, fmt.Errorf("could not open zip file %s: %v", zipname, err)
	}
	defer f.Close()

	for _, file := range f.File {
		rom, ok := roms[file.Name]
		if !ok {				// not in archive
			if isParent {		// if we're in a parent, we already walked over this file in the clone (or this file has a different name in the clone)
				continue
			}
			return false, nil		// otherwise we have a problem
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
		delete(roms, file.Name)		// mark as done
	}

	return true, nil					// all clear on this one
}

func tryParent(which string, roms map[string]*ROM) (bool, error) {
	if optimal[which] == "" {		// if we reached here it should have been found
		return false, nil
	}
	g := games[which]
	good, err := g.checkOneZip(optimal[which], roms, true)
	if err != nil {
		return false, err
	} else if !good {
		return false, nil
	}

	if g.CloneOf != "" {
		good, err := tryParent(g.CloneOf, roms)
		if err != nil {
			return false, err
		}
		if !good {
			return false, nil
		}
	}
	if g.ROMOf != "" && g.ROMOf != g.CloneOf {
		good, err := tryParent(g.ROMOf, roms)
		if err != nil {
			return false, err
		}
		if !good {
			return false, nil
		}
	}

	return true, nil
}

func (g *Game) CheckIn(rompath string) (bool, error) {
	// populate list of ROMs
	var roms = make(map[string]*ROM)
	for i := range g.ROMs {
		if g.ROMs[i].Status != nodump {	// otherwise games with known undumped ROMs will return "not found" because the map never depletes
			// some ROM sets (scregg, for instance) have trailing spaces in the filenames given in he XML file (dc0.c6, in this example)
			// TODO this will also remove leading spaces; is that correct?
			roms[strings.TrimSpace(g.ROMs[i].Name)] = &(g.ROMs[i])
		}
	}

	zipname := g.Filename(rompath)
	good, err := g.checkOneZip(zipname, roms, false)
	if err != nil {
		return false, err
	} else if !good {
		return false, nil
	}

	// TODO eliminate reptition from this and Find()?
	if g.CloneOf != "" {
		good, err := tryParent(g.CloneOf, roms)
		if err != nil {
			return false, err
		}
		if !good {
			return false, nil
		}
	}
	if g.ROMOf != "" && g.ROMOf != g.CloneOf {
		good, err := tryParent(g.ROMOf, roms)
		if err != nil {
			return false, err
		}
		if !good {
			return false, nil
		}
	}

	// if we reached here everything we know about checked out, so if there are any leftover files in the game, that means something is wrong
	return len(roms) == 0, nil
}

func (g *Game) Find() (found bool, err error) {
	// did we find this already?
	if optimal[g.Name] != "" {
		return true, nil
	}

	// find the parents
	if g.CloneOf != "" {
		found, err := games[g.CloneOf].Find()
		if err != nil {
			return false, fmt.Errorf("error finding parent (cloneof) %s: %v", g.CloneOf, err)
		}
		// TODO really bail out?
		if !found {
			return false, err
		}
	}
	if g.ROMOf != "" && g.ROMOf != g.CloneOf {
		found, err := games[g.ROMOf].Find()
		if err != nil {
			return false, fmt.Errorf("error finding parent (romof) %s: %v", g.ROMOf, err)
		}
		// TODO really bail out?
		if !found {
			return false, err
		}
	}

	// go through the directories
	for _, d := range dirs {
		found, err := g.CheckIn(d)
		if err != nil {
			return false, err
		}
		if found {
			optimal[g.Name] = g.Filename(d)
			return true, nil
		}
	}

	// nope
	return false, nil
}
