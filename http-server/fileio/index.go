package fileio

import (
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

// saves the image to the http-server/assets/ directory. Returns relative filepath if success
func ImageUpload(file multipart.File, suffix string) string {
	img := fileToImage(file)
	pathsuffix, dest := createAssetsFile(suffix)
	defer dest.Close()
	write(img, dest)

	return pathsuffix
}

// convert to image.Image
func fileToImage(file multipart.File) image.Image {
	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalln(err)
	}
	return img
}

func createAssetsFile(suffix string) (string, *os.File) {
	pathsuffix := "/assets/" + suffix + ".jpg"
	pathname := filepath.Join(rootDir(), pathsuffix)
	dest, err := os.Create(pathname)
	if err != nil {
		log.Fatalln(err)
	}
	return pathsuffix, dest
}

func write(img image.Image, dest *os.File) {
	err := jpeg.Encode(dest, img, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

// gets the root directory where main.go is running
func rootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}
