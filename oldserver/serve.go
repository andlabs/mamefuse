// 31 december 2012
package main

import (
	// ...
)

type zipRequest struct {
	Game	string
	Return	chan<- string
}

var zipRequests chan zipRequest

var gamesNotDone []string

func startServer() {
	zipRequests = make(chan zipRequest)
	gamesNotDone = make([]string, len(games))
	i := 0
	for _, g := range games {
		gamesNotDone[i] = g.Name
		i++
	}
	go server()
}

func server() {
	for {
		select {
		case req := <-zipRequests:
			found, err := games[req.Game].Find()
			if found {
				req.Return <- games[req.Game].ROMLoc
			} else {
				// TODO handle error
				_ = err
				req.Return <- ""
			}
		default:							// TODO time throttle?
			if len(gamesNotDone) == 0 {		// all done
				continue
			}
			games[gamesNotDone[0]].Find()
			// TODO handle error or failure?
			gamesNotDone = gamesNotDone[:len(gamesNotDone) - 1]
		}
	}
}
