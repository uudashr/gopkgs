package main

import (
	"bufio"
	"flag"
	"fmt"
	"html/template"
	"os"
	"text/tabwriter"

	"github.com/uudashr/gopkgs"
)

var usageInfo = `
Use -format to custom the output using template syntax. The struct being passed to template is:
	type Pkg struct {
		Dir        string // directory containing package sources
		ImportPath string // import path of package in dir
		Name       string // package name
	}
`

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
	tw := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, usageInfo)
}

func init() {
	flag.Usage = usage
}

func main() {
	var (
		flagFormat = flag.String("format", "{{.ImportPath}}", "custom output format")
		flagHelp   = flag.Bool("help", false, "show this message")
	)

	flag.Parse()
	if len(flag.Args()) > 0 || *flagHelp {
		flag.Usage()
		os.Exit(1)
	}

	tpl, err := template.New("out").Parse(*flagFormat)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pkgs, err := gopkgs.Packages()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w := bufio.NewWriter(os.Stdout)
	defer func() {
		if err := w.Flush(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	for _, pkg := range pkgs {
		if err := tpl.Execute(w, pkg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintln(w)
	}
}
