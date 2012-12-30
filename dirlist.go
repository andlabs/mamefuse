// 29 december 2012
package main

import (
	"os"
	"io"
	"bufio"
	"log"
)

var dirs []string

func getDirList(listfile string) {
	_f, err := os.Open(listfile)
	if err != nil {
		log.Fatalf("could not open directory list file %s: %v", listfile, err)
	}
	defer _f.Close()

	f := bufio.NewReader(_f)
	for {
		dir, err := f.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("could not read directory list file %s: %v", listfile, err)
		}
		dirs = append(dirs, dir[:len(dir) - 1])		// drop newline
	}
}
