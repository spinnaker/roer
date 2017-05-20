package cliutil

import "github.com/urfave/cli"

// ContainsAllFlags returns true if cli.Context contains all flagNames
func ContainsAllFlags(cc *cli.Context, flagNames []string) bool {
	z := 0
	for _, n := range flagNames {
		for _, f := range cc.FlagNames() {
			if n == f {
				z++
			}
		}
	}
	return z == len(flagNames)
}
