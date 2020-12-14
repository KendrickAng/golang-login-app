package fileio

import (
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

func ReadPw() string {
	data, err := ioutil.ReadFile(filepath.Join(RootDir() + "/dbPw.txt"))
	if err != nil {
		log.Fatalln(err)
	}
	return string(data)
}

// saves the image to the http-server/assets/ directory. Returns relative filepath if success
func ImageUpload(file multipart.File, suffix string) string {
	img := fileToImage(file)
	pathsuffix, dest := createAssetsFile(suffix)
	defer dest.Close()
	write(img, dest)

	return pathsuffix
}

// returns an absolute image path given User.ProfilePic.
func ImagePath(suffix string) string {
	return RootDir() + suffix
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
	pathname := filepath.Join(RootDir(), pathsuffix)
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
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}
