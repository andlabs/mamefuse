// 29 december 2012
package main

import (
	"fmt"
	"os"
	"code.google.com/p/rsc/fuse"
	"log"
)

// general TODO:
// - be able to ls the ROMs directory
// - switch to rsc's fuse, as Tv` in #go-nuts suggested? from what I can tell this requires a fair bit of work passing up *os.Files

func (g *Game) Find() (found bool, err error) {
	// did we find this already?
	if g.Found {
		return true, nil
	}
	found, err = g.findROMs()
	if err != nil {
		log.Printf("error finding ROMs for game %s: %v\n", g.Name, err)
		return
	} else if !found {
		return
	}
	found, err = g.findCHDs()
	if err != nil {
		log.Printf("error finding CHDs for game %s: %v\n", g.Name, err)
		return
	} else if !found {
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
	fstree := getGames(os.Args[1])
	getDirList(os.Args[2])
	mount, err := fuse.Mount(os.Args[3])
	if err != nil {
		log.Fatalf("error launching FUSE file system: %v", err)
	}
fmt.Println("starting server")
	mount.Serve(fstree)
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
