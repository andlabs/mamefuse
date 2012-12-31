// 29 december 2012
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s mamexml dirlistfile mountpoint\n", os.Args[0])
		os.Exit(1)
	}
	getGames(os.Args[1])
	getDirList(os.Args[2])
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
