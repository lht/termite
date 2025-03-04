package main

import (
	"flag"
	"github.com/hanwen/termite/termite"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	home := os.Getenv("HOME")
	cachedir := flag.String("cachedir",
		filepath.Join(home, ".cache", "termite-master"), "content cache")
	workers := flag.String("workers", "", "comma separated list of worker addresses")
	coordinator := flag.String("coordinator", "localhost:1233",
		"address of coordinator. Overrides -workers")
	socket := flag.String("socket", ".termite-socket", "socket to listen for commands")
	exclude := flag.String("exclude", "/sys,/proc,/dev,/selinux,/cgroup", "prefixes to not export.")
	secretFile := flag.String("secret", "secret.txt", "file containing password.")
	srcRoot := flag.String("sourcedir", "", "root of corresponding source directory")
	jobs := flag.Int("jobs", 1, "number of jobs to run")
	port := flag.Int("port", 1237, "http status port")

	flag.Parse()
	secret, err := ioutil.ReadFile(*secretFile)
	if err != nil {
		log.Fatal("ReadFile", err)
	}

	workerList := strings.Split(*workers, ",")
	excludeList := strings.Split(*exclude, ",")
	c := termite.NewContentCache(*cachedir)
	master := termite.NewMaster(
		c, *coordinator, workerList, secret, excludeList, *jobs)
	master.SetSrcRoot(*srcRoot)
	go master.ServeHTTP(*port)
	master.Start(*socket)
}
