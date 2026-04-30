package utils

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

func PrintSuccess(msg string) { fmt.Printf("%s✔%s  %s\n", green, reset, msg) }
func PrintError(msg string)   { fmt.Fprintf(os.Stderr, "%s✖%s  %s\n", red, reset, msg) }
func PrintInfo(msg string)    { fmt.Printf("%s→%s  %s\n", cyan, reset, msg) }
func PrintWarning(msg string) { fmt.Printf("%s!%s  %s\n", yellow, reset, msg) }

func NewTabWriter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
}
