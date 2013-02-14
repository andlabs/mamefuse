// 29 december 2012
package main

import (
	"fmt"
	"os"
	"github.com/hanwen/go-fuse/fuse"
	"log"
)

// general TODO:
// - be able to ls the ROMs directory

func (g *Game) Find() (found bool, err error) {
fmt.Println("Find()")
	// did we find this already?
	if g.Found {
		return true, nil
	}
	found, err = g.findROMs()
	if !found || err != nil {
		return
	}
	found, err = g.findCHDs()
	if !found || err != nil {
		return
	}
	g.Found = true
	return
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s mamexml dirlistfile mountpoint\n", os.Args[0])
		os.Exit(1)
	}
	getGames(os.Args[1])
	getDirList(os.Args[2])
	fs := fuse.NewPathNodeFs(&fuse.ReadonlyFileSystem{new(mamefuse)}, nil)
	mount, _, err := fuse.MountNodeFileSystem(os.Args[3], fs, nil)
	if err != nil {
		log.Fatalf("error launching FUSE file system: %v", err)
	}
//	fs.Debug = true
//	mount.Debug = true
fmt.Println("starting server")
	mount.Loop()
}

func x() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s mamexml dirlistfile mountpoint\n", os.Args[0])
		os.Exit(1)
	}
	getGames(os.Args[1])
	getDirList(os.Args[2])
//	startServer()
	for _, g := range games {
		fmt.Printf("%12s ", g.Name)
		if g.Found {			// already found (parent)
			fmt.Println(g.ROMLoc)
			continue
		}
		found, err := g.Find()
		if err != nil {
			fmt.Printf("error: %v\n", err)
		} else if !found {
			fmt.Println("not found")
		} else {
			fmt.Println(g.ROMLoc)
		}
	}
}
