package main

import (
	"flag"
	"time"
)

func stringFlagArg(fs *flag.FlagSet, name string, value string) string {
	if !flagWasSet(fs, name) {
		return ""
	}
	return value
}

func durationFlagArg(fs *flag.FlagSet, name string, value time.Duration) string {
	if !flagWasSet(fs, name) {
		return ""
	}
	if value < 0 {
		return ""
	}
	return value.String()
}

func listFlagArg(fs *flag.FlagSet, name string, value []string) []string {
	if !flagWasSet(fs, name) {
		return nil
	}
	return value
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			found = true
		}
	})
	return found
}
