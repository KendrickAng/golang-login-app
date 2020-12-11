package fileio

import (
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"runtime"
)

func ReadPw() string {
	data, err := ioutil.ReadFile(filepath.Join(RootDir() + "/../dbPw.txt"))
	if err != nil {
		log.Fatalln(err)
	}
	return string(data)
}

// gets the root directory where main.go is running
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}
