// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command watchdir issues an inotify watch on a given directory (or a specific
// file within that directory), printing processes matching a given pattern
// whenever a file in the directory is created, deleted, or modified.
//
// Usage: watchdir DIR [FILE] PATTERN
//
// (Where FILE is the short name of a file within DIR.)
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var dir, file, pattern string
	switch len(os.Args) {
	case 3:
		dir = os.Args[1]
		file = "*"
		pattern = os.Args[2]
	case 4:
		dir = os.Args[1]
		file = os.Args[2]
		pattern = os.Args[3]
	default:
		log.Fatalf("usage: %s DIR [FILE] PATTERN", os.Args[0])
	}

	cmd := exec.Command("inotifywait", "-m", "--csv",
		"-e", "create",
		"-e", "delete",
		"-e", "modify",
		"-e", "move",
		"-e", "attrib",
		"-r",
		dir)
	cmd.Stderr = os.Stderr

	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer func() {
		log.Printf("%s:\n\t%v", strings.Join(cmd.Args, " "), cmd.Wait())
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range c {
			cmd.Process.Signal(sig)
		}
	}()

	r := csv.NewReader(out)
	for {
		line, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if len(line) != 3 {
			log.Println("unexpected line: %v", line)
			continue
		}

		event := line[1]
		eFile := line[2]
		if file == "*" || eFile == file {
			pg := exec.Command("pgrep", "-ax", pattern)
			pg.Stderr = new(strings.Builder)

			out, err := pg.Output()
			if err != nil {
				log.Printf("pgrep: %v\n%s", err, pg.Stderr)
			}

			fmt.Printf("*** %s %s\n%s\n", event, eFile, out)
		}
	}
}
