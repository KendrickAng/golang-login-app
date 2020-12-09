package fileio

import (
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
)

// saves the image to the /assets/ directory. Returns relative filepath if success
func ImageUpload(file multipart.File, pwHash string) (string, error) {
	// read
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	// create new file, name is based on password hash (unique)
	path := filepath.Join("../assets/", pwHash)
	dest, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	// write
	_, err = dest.Write(bytes)
	if err != nil {
		return "", err
	}

	return path, nil
}
