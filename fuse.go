// 30 december 2012
package main

import (
	"os"
	"github.com/hanwen/go-fuse/fuse"
	"path/filepath"
	"strings"
	"errors"
	"log"
)

var errNoSuchGame = errors.New("no such game")
var errGameNotFound = errors.New("game not found")

// note to self: this needs to be an embed in a struct as DefaultFileSystem will implement the  methods I don't override here and have them return fuse.ENOSYS
type mamefuse struct {
	fuse.DefaultFileSystem
}

func getgame(gamename string) (*Game, fuse.Status) {
	g, ok := games[gamename]
	if !ok {				// not a valid game
		log.Printf("no such game %s\n", gamename)
		return nil, fuse.EINVAL
	}
	good, err := g.Find()
	if err != nil {
		log.Printf("error finding game %s: %v\n", gamename, err)
		return nil, fuse.EIO
	} else if !good {
		log.Printf("game %s not found\n", gamename)
		return nil, fuse.ENOENT
	}
	return g, fuse.OK
}

func getloopbackfile(filename string) (*fuse.LoopbackFile, fuse.Status) {
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("error opening file %s: %v\n", filename, err)
		return nil, fuse.EIO		// TODO too drastic?
	}
	// according to the go-fuse source (fuse/file.go), fuse.LoopbackFile will take ownership of our *os.FIle, calling Close() on it itself
	return &fuse.LoopbackFile{
		File:	f,
	}, fuse.OK
}

func getattr(filename string) (*fuse.Attr, fuse.Status) {
	stat, err := os.Stat(filename)
	if err != nil {
		log.Printf("error geting stats of file %s: %v\n", filename, err)
		return nil, fuse.EIO		// TODO too drastic?
	}
	return fuse.ToAttr(stat), fuse.OK	// TODO mask out write bits?
}

// to avoid recreating the string each time getchdparts() is called
const sepstr = string(filepath.Separator)

func getchdparts(name string) (gamename string, chdname string) {
	// I know MAME won't hand me pathnames that aren't well-formed but Clean() should make them safe to split like this in general...
	parts := strings.Split(filepath.Clean(name), sepstr)
	if len(parts) < 2 {	// invalid
		return "", ""
	}
	gamename = parts[len(parts) - 2]
	chdname = parts[len(parts) - 1]
	chdname = chdname[:len(chdname) - 4]		// strip .chd so we can find it in the Game structure (the MAME XML file doesn't have the extension)
	return
}

func (fs *mamefuse) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	basename := filepath.Base(name)
	switch filepath.Ext(basename) {
	case ".zip":				// ROM set
		gamename := basename[:len(basename) - 4]
		g, err := getgame(gamename)
		if err != fuse.OK {
			return nil, err
		}
		return getattr(g.ROMLoc)
	case ".chd":
		gamename, chdname := getchdparts(name)
		if gamename == "" {		// we need a game name to disambiguate
			return nil, fuse.ENOENT
		}
		g, err := getgame(gamename)
		if err != fuse.OK {
			return nil, err
		}
		return getattr(g.CHDLoc[chdname])
	default:
		// is it a folder that stores CHDs?
		if _, ok := games[basename]; ok {		// yes
			return &fuse.Attr{
				Mode:	fuse.S_IFDIR | 0755,		// TODO mask out write bits?
			}, fuse.OK
		}
		// no; fall out
	}
	return nil, fuse.ENOENT		// any other file is invalid
}

func (fs *mamefuse) Open(name string, flags uint32, context *fuse.Context) (file fuse.File, code fuse.Status) {
	basename := filepath.Base(name)
	switch filepath.Ext(basename) {
	case ".zip":				// ROM set
		gamename := basename[:len(basename) - 4]
		g, err := getgame(gamename)
		if err != fuse.OK {
			return nil, err
		}
		// TODO worry about closing the file?
		return getloopbackfile(g.ROMLoc)
	case ".chd":				// CHD
		gamename, chdname := getchdparts(name)
		if gamename == "" {		// we need a game name to disambiguate
			return nil, fuse.ENOENT
		}
		g, err := getgame(gamename)
		if err != fuse.OK {
			return nil, err
		}
		// TODO worry about closing the file?
		return getloopbackfile(g.CHDLoc[chdname])
	}
	return nil, fuse.ENOENT		// otherwise 404
}

func (fs *mamefuse) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	// TODO
	return nil, fuse.ENOENT
}
