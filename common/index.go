package common

import "log"

const (
	LOG_PREFIX = "*****"
	LOG_SUFFIX = "*****"
)

func Display(desc string, data ...interface{}) {
	if len(desc) > 0 {
		log.Println(LOG_PREFIX + desc + LOG_SUFFIX)
	}
	if data != nil {
		log.Println(data...)
	}
}
