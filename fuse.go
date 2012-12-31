// 30 december 2012
package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"path/filepath"
)

// note to self: this needs to be an embed in a struct as DefaultFileSystem will implement the  methods I don't override here and have them return fuse.ENOSYS
type mamefuse struct {
	fuse.DefaultFileSystem
}

/*
// TODO; not sure if I need this or not
func (fs *mamefuse) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	// ...
}
*/

func (fs *mamefuse) Open(name string, flags uint32, context *fuse.Context) (file fuse.File, code fuse.Status) {
	basename := filepath.Base(name)
	switch filepath.Ext(basename) {
	case "zip":				// ROM set
		gamename := basename[:len(basename) - 4]
		ret := make(chan string)
		zipRequests <- zipRequest{
			Game:	gamename,
			Return:	ret,
		}
		zipname := <-ret
		close(ret)
		if zipname == "" {		// none given
			// TODO handle error
			return nil, fuse.ENOENT
		}
		f, err := os.Open(zipname)
		if err != nil {
			// TODO report error
			return nil, fuse.EIO	// TODO proper error
		}
		// according to the go-fuse source (fuse/file.go), fuse.LoopbackFile will take ownership of our *os.FIle, calling Close() on it itself
		return fuse.LoopbackFile{f}, nil
	case "chd":				// CHD
		// ...
	case "":					// folder
		// ...
	// TODO root directory?
	}
	return nil, fuse.ENOENT		// otherwise 404
}

func (fs *mamefuse) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	// ...
	return nil, fuse.ENOSYS	// for now
}