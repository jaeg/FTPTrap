package main

import (
	"math/rand"
	"os"
	"time"
)

//JunkFile represents a junk file presented by the server.
type JunkFile struct {
	FileName string
	Content  string
}

//Name ..
func (j JunkFile) Name() string {
	return j.FileName
}

//Size ..
func (j JunkFile) Size() int64 {
	return rand.Int63()
}

//Mode ..
func (j JunkFile) Mode() os.FileMode {
	return 777
}

//ModTime ..
func (j JunkFile) ModTime() time.Time {
	return time.Now()
}

//IsDir ..
func (j JunkFile) IsDir() bool {
	return false
}

//Sys ..
func (j JunkFile) Sys() interface{} {
	return nil
}
