package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nlink-jp/eml-to-jsonl/internal/parser"
)

var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	pretty := flag.Bool("pretty", false, "pretty-print JSON (default: JSONL)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: eml-to-jsonl [flags] [file.eml | dir/ ...]\n\n")
		fmt.Fprintf(os.Stderr, "Parses EML files and outputs structured JSONL to stdout.\n")
		fmt.Fprintf(os.Stderr, "Reads from stdin when no arguments are given.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	enc := newEncoder(os.Stdout, *pretty)
	hadError := false

	if flag.NArg() == 0 {
		if err := processReader(os.Stdin, "stdin", enc); err != nil {
			fmt.Fprintf(os.Stderr, "error: stdin: %v\n", err)
			hadError = true
		}
	} else {
		for _, arg := range flag.Args() {
			if err := processArg(arg, enc); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
				hadError = true
			}
		}
	}

	if hadError {
		os.Exit(1)
	}
}

// processArg handles a single CLI argument: a file, a directory, or a glob.
func processArg(arg string, enc *encoder) error {
	info, err := os.Stat(arg)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return processDir(arg, enc)
	}
	return processFile(arg, enc)
}

// processDir walks a directory and processes all *.eml files found directly
// in that directory (non-recursive, matching the documented behaviour).
func processDir(dir string, enc *encoder) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.eml"))
	if err != nil {
		return err
	}
	hadError := false
	for _, path := range matches {
		if err := processFile(path, enc); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", path, err)
			hadError = true
		}
	}
	if hadError {
		return fmt.Errorf("one or more files in %s failed to parse", dir)
	}
	return nil
}

// processFile opens an EML file and processes it.
func processFile(path string, enc *encoder) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return processReader(f, path, enc)
}

// processReader parses one EML message from r and writes it to enc.
func processReader(r io.Reader, source string, enc *encoder) error {
	email, err := parser.Parse(r, source)
	if err != nil {
		return err
	}
	return enc.write(email)
}

// encoder wraps json.Encoder and optionally pretty-prints output.
type encoder struct {
	w      io.Writer
	pretty bool
}

func newEncoder(w io.Writer, pretty bool) *encoder {
	return &encoder{w: w, pretty: pretty}
}

func (e *encoder) write(v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if e.pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := e.w.Write(buf.Bytes())
	return err
}
