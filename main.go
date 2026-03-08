package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SQLek/wihajster/internal/lexer"
	"github.com/SQLek/wihajster/internal/parser"
	"github.com/SQLek/wihajster/internal/sema"
	"github.com/SQLek/wihajster/internal/tac"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("wihajster", flag.ContinueOnError)
	fs.SetOutput(stderr)

	outPath := fs.String("o", "", "write TAC output to file (default: stdout)")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [-o output.tac] <input.c>\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one input C file")
	}

	inPath := fs.Arg(0)
	in, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("open input %q: %w", inPath, err)
	}
	defer in.Close()

	lex := lexer.NewLexer(in)
	tu, err := parser.Parse(lex)
	if err != nil {
		return err
	}

	mod, err := sema.Lower(tu)
	if err != nil {
		return err
	}

	out := stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			return fmt.Errorf("create output %q: %w", *outPath, err)
		}
		defer f.Close()
		out = f
	}

	if err := tac.WriteModule(out, mod); err != nil {
		return fmt.Errorf("write TAC: %w", err)
	}

	return nil
}
