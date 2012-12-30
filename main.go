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
	openMAMEXML(os.Args[1])
	getDirList(os.Args[2])
	for g, done := getNextGame(); !done; g, done = getNextGame() {
		var err error

		fmt.Printf("%12s ", g.Name)
		found := false
		for _, d := range dirs {
			found, err = g.CheckIn(d)
			if err != nil {
				fmt.Printf("error: %v\n", err)
				found = true
				break
			}
			if found {
				fmt.Println(g.Filename(d))
				break			}
		}
		if !found {
			fmt.Println("not found")
		}
	}
}
