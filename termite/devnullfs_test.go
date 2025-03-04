package termite

import (
	"github.com/hanwen/go-fuse/fuse"
	"os"
	"io/ioutil"
	"testing"
)

func setupDevNullFs() (wd string, clean func()) {
	fs := NewDevnullFs()
	mountPoint := fuse.MakeTempDir()
	state, _, err := fuse.MountPathFileSystem(mountPoint, fs, nil)
	if err != nil {
		panic(err)
	}

	state.Debug = true
	go state.Loop(false)
	return mountPoint, func() {
		state.Unmount()
		os.RemoveAll(mountPoint)
	}
}

func TestDevNullFs(t *testing.T) {
	wd, clean := setupDevNullFs()
	defer clean()

	err := ioutil.WriteFile(wd+"/null", []byte("ignored"), 0644)
	if err != nil {
		t.Error(err)
	}

	result, err := ioutil.ReadFile(wd + "/null")
	if err != nil {
		t.Error(err)
	}
	if len(result) > 0 {
		t.Error("Should have 0 length read.")
	}
}
