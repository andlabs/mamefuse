// 29 december 2012
package main

import (
	"fmt"
	"os"
	"io"
	"path/filepath"
	"encoding/hex"
	"bytes"
	"strings"
	"log"
)

// TODO:
// - select one constructor syntax for the maps?
// - condense map[string]*ROM into a named type?
// - not have sha1hash be global?

func sha1check_chd(f *os.File, expectstring string) (bool, error) {
	expected, err := hex.DecodeString(expectstring)
	if err != nil {
		log.Fatalf("hex decode error reading sha1 (%q): %v", expectstring, err)
	}

	sha1hash.Reset()
	n, err := io.Copy(sha1hash, f)
	if err != nil {
		return false, fmt.Errorf("could not read given zip file entry: %v", err)
	}
	// TODO figure out how to check filesize?
	_ = n

	return bytes.Equal(expected, sha1hash.Sum(nil)), nil
}

// TODO change filename_ROM to not be part of Game as well
func filename_CHD(rompath string, gamename string, CHDname string) string {
	return filepath.Join(rompath, gamename, CHDname + ".chd")
}

func (g *Game) checkCHDIn(rompath string, chd *ROM) (bool, string, error) {
	try := func(dir string) (bool, string, error) {
		fn := filename_CHD(rompath, dir, chd.Name)
		file, err := os.Open(fn)
		if os.IsNotExist(err) {
			return false, "", nil
		} else if err != nil {
			return false, "", fmt.Errorf("could not open CHD file %s: %v", fn, err)
		}
		good, err := sha1check_chd(file, chd.SHA1)
		file.Close()
		if err != nil {
			return false, "", fmt.Errorf("could not calculate SHA-1 sum of CHD %s: %v", fn, err)
		}
		if !good {
			return false, "", nil
		}
		return true, fn, nil
	}

	// first try the game
	found, path, err := try(g.Name)
	if err != nil {
		return false, "", err
	}
	if found {
		return true, path, nil
	}

	// then its parents
	for _, p := range g.Parents {
		found, path, err := try(p)
		if err != nil {
			return false, "", err
		}
		if found {
			return true, path, nil
		}
	}

	// nope
	return false, "", nil
}

// remove all CHDs belonging to this set and its parents from the list
func (g *Game) strikeCHDs(chds map[string]*ROM) {
	for _, rom := range g.CHDs {
		delete(chds, rom.Name)
	}
	for _, parent := range g.Parents {
		games[parent].strikeCHDs(chds)
	}
}

func (g *Game) findCHDs() (found bool, err error) {
	// did we find this already?
	if g.Found {
		return true, nil
	}

	// populate list of CHDs
	var chds = make(map[string]*ROM)
	for i := range g.CHDs {
		if g.CHDs[i].Status != nodump {		// otherwise games with known undumped ROMs will return "not found" because the map never depletes
			// some ROM sets (scregg, for instance) have trailing spaces in the filenames given in he XML file (dc0.c6, in this example)
			// TODO this will also remove leading spaces; is that correct?
			chds[strings.TrimSpace(g.CHDs[i].Name)] = &(g.CHDs[i])
		}
	}

	// find the parents and remove their CHDs rom the list
	for _, parent := range g.Parents {
		found, err := games[parent].Find()
		if err != nil {
			return false, fmt.Errorf("error finding parent %s: %v", parent, err)
		}
		// TODO really bail out?
		if !found {
			return false, err
		}
		games[parent].strikeCHDs(chds)
	}

	if len(chds) == 0 {		// no CHDs left to check (either has no CHDs or we are done)
		g.Found = true
		return true, nil
	}

	// go through the directories, finding the right file
	n := len(chds)
	for _, d := range dirs {
		for name, chd := range chds {
			found, path, err := g.checkCHDIn(d, chd)
			if err != nil {
				return false, err
			}
			if found {
				g.CHDLoc[name] = path
				n--
				break
			}
		}
	}

	if n == 0 {		// all found!
		g.Found = true
		return true, nil
	}

	// nope
	return false, nil
}
