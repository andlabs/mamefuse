// 30 december 2012
package main

import (
	"os"
	"code.google.com/p/rsc/fuse"
	"path/filepath"
)

// TODO:
// - is fuse.Tree read-only?
// - is ls trying to open all the files? because an ls on the mounted directory takes forever

func (g *Game) AddToTree(t *fuse.Tree) {
	t.Add(g.Name + ".zip", NewROMFile(g))
	for _, c := range g.CHDs {
		t.Add(filepath.Join(g.Name, c.Name + ".chd"), NewCHDFile(g, c.Name))
	}
}

// generic file node and handle; embedded by ROMNode and CHDNode to get the job done
type FUSEFile struct {
	size		uint64
	f		*os.File
}

func (f *FUSEFile) Attr() fuse.Attr {
	return fuse.Attr{
		Mode:	0444,
		Size:		f.size,
	}
}

func (f *FUSEFile) open(filename string) fuse.Error {
	var err error

	f.f, err = os.Open(filename)
	if err != nil {
		return fuse.EIO
	}
	if f.size == 0 {			// try to get size
		s, err := f.f.Stat()
		if err != nil {		// if we failed, we have an issue
			f.f.Close()		// TODO let it slide?
			return fuse.EIO
		}
		f.size = uint64(s.Size())		// int64 -> uint64 should be safe
	}
	return nil
}

func (f *FUSEFile) Read(req *fuse.ReadRequest, resp *fuse.ReadResponse, intr fuse.Intr) fuse.Error {
	// TODO check to see if opened?
	resp.Data = make([]byte, req.Size)
	_, err := f.f.ReadAt(resp.Data, req.Offset)
	if err != nil {
		return fuse.EIO
	}
	return nil
}

func (f *FUSEFile) Release(*fuse.ReleaseRequest, fuse.Intr) fuse.Error {
	// TODO check to see if opened?
	f.f.Close()
	return nil
}

type ROMFile struct {
	g		*Game
	*FUSEFile
}

// because the *FUSEFile embed won't allocate itself
func NewROMFile(g *Game) *ROMFile {
	return &ROMFile{
		g:		g,
		FUSEFile:	new(FUSEFile),
	}
}

// TODO DRY?
func (r *ROMFile) Open(req *fuse.OpenRequest, resp *fuse.OpenResponse, intr fuse.Intr) (h fuse.Handle, ferr fuse.Error) {
	// TODO uint32 conversion here safe? should I just use the syscall ones instead? FUSE documentation says the values should match...
	if (req.Flags & uint32(os.O_WRONLY | os.O_RDWR)) != 0 {		// ban writes
		return nil, fuse.EPERM
	}
	found, err := r.g.Find()
	if !found || err != nil {		// TODO report error somehow
		return nil, fuse.ENOENT
	}
	ferr = r.open(r.g.ROMLoc)
	if ferr != nil {
		return nil, ferr
	}
	return r, nil
}

type CHDFile struct {
	g		*Game
	name	string
	*FUSEFile
}

// because the *FUSEFile embed won't allocate itself
func NewCHDFile(g *Game, name string) *CHDFile {
	return &CHDFile{
		g:		g,
		name:	name,
		FUSEFile:	new(FUSEFile),
	}
}

// TODO DRY?
func (r *CHDFile) Open(req *fuse.OpenRequest, resp *fuse.OpenResponse, intr fuse.Intr) (h fuse.Handle, ferr fuse.Error) {
	if (req.Flags & uint32(os.O_WRONLY | os.O_RDWR)) != 0 {		// ban writes
		return nil, fuse.EPERM
	}
	found, err := r.g.Find()
	if !found || err != nil {
		return nil, fuse.ENOENT
	}
	ferr = r.open(r.g.CHDLoc[r.name])
	if ferr != nil {
		return nil, ferr
	}
	return r, nil
}
