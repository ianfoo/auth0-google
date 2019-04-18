package main

import (
	"fmt"
	"log"
	"os"
)

// Panic if error.
func Must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

func getFromFlagOrEnv(val, envVar, humanReadable string) string {
	if val != "" {
		return val
	}
	if envVal := os.Getenv(envVar); envVal != "" {
		return envVal
	}
	bail(fmt.Errorf("%s is required", humanReadable))
	return ""
}

func bail(err error) {
	if err == nil {
		return
	}
	log.SetFlags(0)
	log.SetPrefix(os.Args[0] + ": ")
	log.Fatal(err)
}
