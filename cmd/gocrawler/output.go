package main

import (
	"fmt"

	"github.com/pepetka/gocrawler"
)

func printResults(results []gocrawler.Result) {
	for _, wr := range results {
		fmt.Println(wr)
	}
}
