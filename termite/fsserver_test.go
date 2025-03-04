package termite

// This file tests both rpcfs and fsserver by having them talk over a
// socketpair.

import (
	"github.com/hanwen/go-fuse/fuse"
	"io/ioutil"
	"log"
	"os"
	"rpc"
	"testing"
)

func TestFsServerCache(t *testing.T) {
	log.Println("TestFsServerCache")
	tmp, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(tmp)

	orig := tmp + "/orig"
	srvCache := tmp + "/server-cache"

	err := os.Mkdir(orig, 0700)
	if err != nil {
		t.Fatal(err)
	}

	content := "hello"
	err = ioutil.WriteFile(orig+"/file.txt", []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cache := NewContentCache(srvCache)
	server := NewFsServer("/", cache, nil)

	server.refreshAttributeCache(orig)
	if len(server.attrCache) > 0 {
		t.Errorf("cache not empty? %#v", server.attrCache)
	}

	server.oneGetAttr(orig)
	server.oneGetAttr(orig+"/file.txt")

	if len(server.attrCache) != 2 {
		t.Errorf("cache should have 2 entries, got %#v", server.attrCache)
	}
	name := orig + "/file.txt"
	attr, ok := server.attrCache[name]
	if !ok || !attr.FileInfo.IsRegular() || attr.FileInfo.Size != int64(len(content)) {
		t.Errorf("entry for %q unexpected: %v %#v", name, ok, attr)
	}

	newName := orig + "/new.txt"
	err = os.Rename(name, newName)
	if err != nil {
		t.Fatal(err)
	}

	server.refreshAttributeCache(orig)
	attr, ok = server.attrCache[name]
	if !ok || attr.Status.Ok() {
		t.Errorf("after rename: entry for %q unexpected: %v %#v", name, ok, attr)
	}
}

func TestRpcFS(t *testing.T) {
	tmp, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(tmp)

	mnt := tmp + "/mnt"
	orig := tmp + "/orig"
	srvCache := tmp + "/server-cache"
	clientCache := tmp + "/client-cache"

	os.Mkdir(mnt, 0700)
	os.Mkdir(orig, 0700)
	os.Mkdir(orig+"/subdir", 0700)
	content := "hello"
	err := ioutil.WriteFile(orig+"/file.txt", []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cache := NewContentCache(srvCache)
	server := NewFsServer(orig, cache, []string{})

	l, r, err := fuse.Socketpair("unix")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	defer r.Close()

	rpcServer := rpc.NewServer()
	rpcServer.Register(server)
	go rpcServer.ServeConn(l)

	rpcClient := rpc.NewClient(r)
	fs := NewRpcFs(rpcClient, NewContentCache(clientCache))

	state, _, err := fuse.MountPathFileSystem(mnt, fs, nil)
	state.Debug = true
	if err != nil {
		t.Fatal("Mount", err)
	}
	defer func() {
		log.Println("unmounting")
		err := state.Unmount()
		if err == nil {
			os.RemoveAll(tmp)
		}
	}()

	go state.Loop(false)

	fi, err := os.Lstat(mnt + "/subdir")
	if fi == nil || !fi.IsDirectory() {
		t.Fatal("subdir stat", fi, err)
	}

	c, err := ioutil.ReadFile(mnt + "/file.txt")
	if err != nil || string(c) != "hello" {
		t.Error("Readfile", c)
	}

	entries, err := ioutil.ReadDir(mnt)
	if err != nil || len(entries) != 2 {
		t.Error("Readdir", err, entries)
	}

	// This test implementation detail - should be separate?
	storedHash := server.hashCache["/file.txt"]
	if storedHash == "" || string(storedHash) != string(md5str(content)) {
		t.Errorf("cache error %x (%v)", storedHash, storedHash)
	}

	newData := []FileAttr{
		FileAttr{
			Path: "/file.txt",
			Hash: md5str("somethingelse"),
		},
	}
	server.updateFiles(newData)
	storedHash = server.hashCache["/file.txt"]
	if storedHash == "" || storedHash != newData[0].Hash {
		t.Errorf("cache error %x (%v)", storedHash, storedHash)
	}
}
