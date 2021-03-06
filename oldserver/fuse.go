// 30 december 2012
package main

import (
	"os"
	"github.com/hanwen/go-fuse/fuse"
	"path/filepath"
	"strings"
	"errors"
)

var errNoSuchGame = errors.New("no such game")
var errGameNotFound = errors.New("game not found")

// note to self: this needs to be an embed in a struct as DefaultFileSystem will implement the  methods I don't override here and have them return fuse.ENOSYS
type mamefuse struct {
	fuse.DefaultFileSystem
}

func getgame(gamename string) (*Game, error) {
	g, ok := games[gamename]
	if !ok {				// not a valid game
		return nil, errNoSuchGame
	}
//	ret := make(chan string)
//	zipRequests <- zipRequest{
//		Game:	gamename,
//		Return:	ret,
//	}
//	zipname := <-ret
//	close(ret)
//	if zipname == "" {		// none given
	good, err := g.Find()
	if err != nil {
		return nil, err
	}
	if !good {
		return nil, errGameNotFound
	}
	return g, nil
}

func getgame_fuseerr(gamename string) (*Game, fuse.Status) {
	g, err := getgame(gamename)
	if err == errNoSuchGame {
		return nil, fuse.EINVAL
	} else if err == errGameNotFound {
		return nil, fuse.ENOENT
	} else if err != nil {
		// TODO report error
		return nil, fuse.EIO
	}
	return g, fuse.OK
}

func getloopbackfile(filename string) (*fuse.LoopbackFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// according to the go-fuse source (fuse/file.go), fuse.LoopbackFile will take ownership of our *os.FIle, calling Close() on it itself
	return &fuse.LoopbackFile{
		File:	f,
	}, nil
}

func getloopbackfile_fuseerr(filename string) (*fuse.LoopbackFile, fuse.Status) {
	loopfile, err := getloopbackfile(filename)
	if err != nil {
		// TODO report error
		return nil, fuse.EIO
	}
	return loopfile, fuse.OK
}

func getattr(filename string) (*fuse.Attr, fuse.Status) {
	stat, err := os.Stat(filename)
	if err != nil {
		// TODO report error
		return nil, fuse.EIO
	}
	return fuse.ToAttr(stat), fuse.OK
}

// to avoid recreating the string each time getchdparts() is called
var sepstr = string(filepath.Separator)

func getchdparts(name string) (gamename string, chdname string) {
	// I know MAME won't hand me pathnames that aren't well-formed but Clean() should make them safe to split like this in general...
	parts := strings.Split(filepath.Clean(name), sepstr)
	if len(parts) < 2 {	// invalid
		return "", ""
	}
	gamename = parts[len(parts) - 2]
	chdname = parts[len(parts) - 1]
	chdname = chdname[:len(chdname) - 4]		// strip .chd
	return
}

func (fs *mamefuse) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	basename := filepath.Base(name)
	switch filepath.Ext(basename) {
	case ".zip":				// ROM set
		gamename := basename[:len(basename) - 4]
		g, err := getgame_fuseerr(gamename)
		if err != fuse.OK {
			return nil, err
		}
//		return getattr(g.ROMLoc)
		// TODO merely returning getattr() always results in
		// 2012/12/31 12:13:27 writer: Write/Writev failed, err: 22=invalid argument. opcode: LOOKUP
		// but this works
		a, err := getattr(g.ROMLoc)
		return a, err
	case ".chd":
		gamename, chdname := getchdparts(name)
		if gamename == "" {		// we need a game name to disambiguate
			return nil, fuse.ENOENT
		}
		g, err := getgame_fuseerr(gamename)
		if err != fuse.OK {
			return nil, err
		}
//		return getattr(g.ROMLoc)
		// TODO merely returning getattr() always results in
		// 2012/12/31 12:13:27 writer: Write/Writev failed, err: 22=invalid argument. opcode: LOOKUP
		// but this works
		a, err := getattr(g.CHDLoc[chdname])
		return a, err
	default:
		// is it a folder that stores CHDs?
		if _, ok := games[basename]; ok {		// yes
			return &fuse.Attr{
				Mode:	fuse.S_IFDIR | 0755,
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
		g, err := getgame_fuseerr(gamename)
		if err != fuse.OK {
			return nil, err
		}
		// TODO worry about closing the file?
		return getloopbackfile_fuseerr(g.ROMLoc)
	case ".chd":				// CHD
		gamename, chdname := getchdparts(name)
		if gamename == "" {		// we need a game name to disambiguate
			return nil, fuse.ENOENT
		}
		g, err := getgame_fuseerr(gamename)
		if err != fuse.OK {
			return nil, err
		}
		// TODO worry about closing the file?
		return getloopbackfile_fuseerr(g.CHDLoc[chdname])
	case "":					// folder
		// ...
	// TODO root directory?
	}
	return nil, fuse.ENOENT		// otherwise 404
}

func (fs *mamefuse) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	// TODO
	return nil, fuse.ENOENT
}