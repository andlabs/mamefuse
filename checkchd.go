// 29 december 2012
package main

import (
	"fmt"
	"os"
	"io"
	"path/filepath"
	"encoding/hex"
	"encoding/binary"
	"bytes"
	"strings"
	"log"
)

// TODO:
// - select one constructor syntax for the maps?

type CHDs map[string]*CHD

var sha1Off = map[uint32]int64{
	// right now using standard; comment has parent (raw if also available)
	3:	80,		// 100
	4:	48,		// 68 (raw 88)
	5:	84,		// 104 (raw 64)
}

const versionFieldOff = 12

func sha1check_chd(f *os.File, expectstring string) (bool, error) {
	expected, err := hex.DecodeString(expectstring)
	if err != nil {
		log.Fatalf("hex decode error reading sha1 (%q): %v", expectstring, err)
	}

	var version uint32
	var sha1 [20]byte

	_, err = f.Seek(versionFieldOff, 0)
	if err != nil {
		return false, fmt.Errorf("seek in CHD to find version number failed: %v", err)
	}
	err = binary.Read(f, binary.BigEndian, &version)
	if err != nil {
		return false, fmt.Errorf("read version number from CHD failed: %v", err)
	}

	if sha1Off[version] == 0 {
		return false, fmt.Errorf("invalid CHD version %d", version)
	}
	_, err = f.Seek(sha1Off[version], 0)
	if err != nil {
		return false, fmt.Errorf("seek in CHD to get SHA-1 sum failed: %v", err)
	}
	_, err = io.ReadFull(f, sha1[:])
	if err != nil {
		return false, fmt.Errorf("read of SHA-1 failed: %v", err)
	}

	return bytes.Equal(expected, sha1[:]), nil
}

func filename_CHD(rompath string, gamename string, CHDname string) string {
	return filepath.Join(rompath, gamename, CHDname + ".chd")
}

func (g *Game) checkCHDIn(rompath string, chd *CHD) (bool, string, error) {
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
func (g *Game) strikeCHDs(chds CHDs) {
	for _, rom := range g.CHDs {
		delete(chds, rom.Name)
	}
	for _, parent := range g.Parents {
		games[parent].strikeCHDs(chds)
	}
}

func (g *Game) findCHDs() (found bool, err error) {
	g.CHDLoc = map[string]string{}

	// populate list of CHDs
	var chds = make(CHDs)
	for i := range g.CHDs {
		if g.CHDs[i].Status != nodump {		// otherwise games with known undumped CHDs will return "not found" because the map never depletes
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
		if !found {
			return false, nil		// TODO return parent not found as an error?
		}
		games[parent].strikeCHDs(chds)
	}

	if len(chds) == 0 {		// no CHDs left to check (either has no CHDs or we are done)
		return true, nil
	}

	// go through the directories, finding the right file
	n := len(chds)
	for name, chd := range chds {
		for _, d := range dirs {
			found, path, err := g.checkCHDIn(d, chd)
			if err != nil {
				return false, err
			}
			if found {
				g.CHDLoc[name] = path
				n--
				break		// found it in this dir; stop scanning dirs and go to the next CHD
			}
		}
	}

	if n == 0 {		// all found!
		return true, nil
	}

	// nope
	return false, nil
}
