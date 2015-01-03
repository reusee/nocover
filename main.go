package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("usage: %s [profile file]", os.Args[0])
		return
	}
	content, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	entries := strings.Split(string(content), "\n")[1:]
	fileLines := make(map[string][]string)
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if !strings.HasSuffix(entry, " 0") {
			continue
		}
		parts := strings.Split(entry, ":")
		path := parts[0]
		if path[0] == '_' {
			path = path[1:]
		} else {
			path = filepath.Join(os.Getenv("GOPATH"), "src", path)
		}
		lines, ok := fileLines[path]
		if !ok {
			content, err = ioutil.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}
			lines = strings.Split(string(content), "\n")
			fileLines[path] = lines
		}
		parts = strings.SplitN(parts[1], ".", 2)
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Fatal(err)
		}
		if strings.HasSuffix(strings.TrimSpace(lines[start-1]), "NOCOVER") {
			continue
		}
		fmt.Printf("%s : %d\n", path, start)
	}
}
