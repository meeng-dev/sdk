// Command fakeapi serves a fake of meeng's public API, built from the API's
// contract (see package fakeapi), for the contract tests of the SDKs.
//
//	fakeapi -spec openapi.json -url-file /tmp/fake.url   # serve until killed
//	fakeapi -coverage http://127.0.0.1:4010              # exit 1 if an operation was never called
//	fakeapi -reset http://127.0.0.1:4010                 # forget the requests and the coverage
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/meeng-dev/sdk/fakeapi"
)

func main() {
	spec := flag.String("spec", "openapi.json", "the contract to serve")
	addr := flag.String("addr", "127.0.0.1:0", "where to listen; a loopback address")
	urlFile := flag.String("url-file", "", "a file to write the address in once the server listens")
	coverage := flag.String("coverage", "", "the address of a running fake: exit 1 when an operation was never answered")
	reset := flag.String("reset", "", "the address of a running fake: forget its requests")
	flag.Parse()
	switch {
	case *coverage != "":
		os.Exit(report(*coverage))
	case *reset != "":
		request(http.MethodDelete, strings.TrimRight(*reset, "/")+"/__requests")
		return
	}
	raw, err := os.ReadFile(*spec)
	if err != nil {
		fail(err)
	}
	server, err := fakeapi.New(raw)
	if err != nil {
		fail(err)
	}
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		fail(err)
	}
	if ip, ok := listener.Addr().(*net.TCPAddr); !ok || !ip.IP.IsLoopback() {
		fail(fmt.Errorf("fakeapi: %s is not a loopback address: the fake has no authentication of its own", listener.Addr()))
	}
	address := "http://" + listener.Addr().String()
	fmt.Println("fakeapi: serving", *spec, "at", address)
	if *urlFile != "" {
		if err := os.WriteFile(*urlFile, []byte(address+"\n"), 0o600); err != nil {
			fail(err)
		}
	}
	fail(http.Serve(listener, server))
}

func request(method, url string) []byte {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fail(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body
}

// report says which operations were never answered.
func report(base string) int {
	var got struct {
		Operations []string `json:"operations"`
		Missing    []string `json:"missing"`
	}
	if err := json.Unmarshal(request(http.MethodGet, strings.TrimRight(base, "/")+"/__coverage"), &got); err != nil {
		fail(err)
	}
	fmt.Printf("fakeapi: %d of %d operations answered\n", len(got.Operations)-len(got.Missing), len(got.Operations))
	for _, missing := range got.Missing {
		fmt.Println("fakeapi: never called:", missing)
	}
	if len(got.Missing) > 0 {
		return 1
	}
	return 0
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
