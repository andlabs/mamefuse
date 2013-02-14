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

// TODO:
// - select one constructor syntax for the maps?
// - condense map[string]*ROM into a named type?

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

	var sha1hash = sha1.New()

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

func (g *Game) filename_ROM(rompath string) string {
	return filepath.Join(rompath, g.Name + ".zip")
}

func (g *Game) checkIn(rompath string, roms map[string]*ROM) (bool, error) {
	zipname := g.filename_ROM(rompath)
	f, err := zip.OpenReader(zipname)
	if os.IsNotExist(err) {		// if the file does not exist, try the next rompath
		return false, nil
	}
	if err != nil {			// something different happened
		return false, fmt.Errorf("could not open zip file %s: %v", zipname, err)
	}
	defer f.Close()

	// true values will be written to this as we find valid ROMs
	// if the length of this does not equal the length of roms when we're done; we missed something and therefore something else is wrong
	var found = map[string]bool{}

	for _, file := range f.File {
		rom, ok := roms[file.Name]
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
		found[file.Name] = true		// mark as done
	}

	// if we reached here everything we know about checked out, so if there are any leftover files in the game, that means something is wrong
	return len(roms) == len(found), nil
}

// remove all ROMs belonging to this set and its parents from the list
func (g *Game) strikeROMs(roms map[string]*ROM) {
	for _, rom := range g.ROMs {
		delete(roms, rom.Name)
	}
	for _, parent := range g.Parents {
		games[parent].strikeROMs(roms)
	}
}

func (g *Game) findROMs() (found bool, err error) {
	// populate list of ROMs
	var roms = make(map[string]*ROM)
	for i := range g.ROMs {
		if g.ROMs[i].Status != nodump {	// otherwise games with known undumped ROMs will return "not found" because the map never depletes
			// some ROM sets (scregg, for instance) have trailing spaces in the filenames given in he XML file (dc0.c6, in this example)
			// TODO this will also remove leading spaces; is that correct?
			roms[strings.TrimSpace(g.ROMs[i].Name)] = &(g.ROMs[i])
		}
	}

	// find the parents and remove their ROMs rom the list
	for _, parent := range g.Parents {
		found, err := games[parent].Find()
		if err != nil {
			return false, fmt.Errorf("error finding parent %s: %v", parent, err)
		}
		// TODO really bail out?
		if !found {
			return false, err
		}
		games[parent].strikeROMs(roms)
	}

	if len(roms) == 0 {		// no ROMs left to check (either has no ROMs or is just a CHD after BIOSes)
		return true, nil
	}

	// go through the directories, finding the right file
	for _, d := range dirs {
		found, err := g.checkIn(d, roms)
		if err != nil {
			return false, err
		}
		if found {
			g.ROMLoc = g.filename_ROM(d)
			return true, nil
		}
	}

	// nope
	return false, nil
}
