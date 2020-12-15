package profiling

import (
	"example.com/kendrick/common"
	"example.com/kendrick/fileio"
	"log"
	"os"
	"path"
)

const IS_PROFILING = true

var LOGIN_LOGFILE string = path.Join(fileio.RootDir(), "loginsLog.txt")

// clears all log files
func InitLogFiles() {
	deleteFile(LOGIN_LOGFILE)
	createFile(LOGIN_LOGFILE)
	common.Print("INIT FRESH LOG FILE " + LOGIN_LOGFILE)
}

func RecordLogin(str string) {
	if !IS_PROFILING {
		return
	}
	writeFile(LOGIN_LOGFILE, str)
}

func createFile(filePath string) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if isError(err) {
			return
		}
		defer file.Close()
	}
}

func writeFile(filePath string, content string) {
	// Open file with READ and WRITE permissions
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, 0644)
	if isError(err) {
		return
	}
	defer file.Close()

	// Write text
	_, err = file.WriteString(content)
	if isError(err) {
		return
	}

	// Save file changes
	err = file.Sync()
	if isError(err) {
		return
	}
}

func deleteFile(filePath string) {
	err := os.Remove(filePath)
	if isError(err) {
		return
	}
}

func isError(err error) bool {
	if err != nil {
		log.Println(err.Error())
	}
	return err != nil
}
