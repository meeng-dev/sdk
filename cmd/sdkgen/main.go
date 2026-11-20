// Command sdkgen writes meeng's SDKs from the public API's contract.
//
//	go run ./cmd/sdkgen                                  # every language
//	go run ./cmd/sdkgen -language go                     # one
//	go run ./cmd/sdkgen -sync ../meeng.dev/docs/contracts/api.openapi.json
//
// -sync copies the contract meeng.dev exports into openapi.json first, so the
// contract the SDKs were written from is kept beside them. The copy is the
// app's contract byte for byte, except for a sentence of its description that
// names another product (generator.SanitizeContract).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meeng-dev/sdk/generator"
)

func main() {
	spec := flag.String("spec", "openapi.json", "the public API's contract")
	sync := flag.String("sync", "", "a contract to copy into -spec before generating")
	language := flag.String("language", "", "one language; every language when empty")
	out := flag.String("out", ".", "the directory that holds VERSION and <language>/")
	flag.Parse()
	if *sync != "" {
		raw, err := os.ReadFile(*sync)
		if err == nil {
			raw, err = generator.SanitizeContract(raw)
		}
		if err == nil {
			err = os.WriteFile(*spec, raw, 0o644)
		}
		if err != nil {
			fail("sdkgen: sync the contract:", err)
		}
	}
	version, err := os.ReadFile(filepath.Join(*out, "VERSION"))
	if err != nil {
		fail("sdkgen: read the SDK version:", err)
	}
	ir, err := generator.Load(*spec, strings.TrimSpace(string(version)))
	if err != nil {
		fail(err)
	}
	languages := []string{*language}
	if *language == "" {
		languages = languages[:0]
		for name := range generator.Languages {
			languages = append(languages, name)
		}
		sort.Strings(languages)
	}
	for _, name := range languages {
		if err := generator.Generate(ir, name, filepath.Join(*out, name)); err != nil {
			fail(err)
		}
		fmt.Println("sdkgen: wrote", filepath.Join(*out, name))
		if name == "dart" {
			formatDart(filepath.Join(*out, name))
		}
	}
}

// formatDart runs `dart format` over the generated Dart, which is written
// unformatted: the formatter's output is the one the committed files and the
// gate (`dart format --set-exit-if-changed`) agree on. Without the Dart SDK on
// the PATH the files are left as written, and the gate will say so.
func formatDart(dir string) {
	dart, err := exec.LookPath("dart")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sdkgen: dart is not on the PATH, so the Dart SDK is left unformatted")
		return
	}
	cmd := exec.Command(dart, "format", "--language-version=3.8", dir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail("sdkgen: dart format:", err)
	}
}

func fail(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}
