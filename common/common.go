package common

import (
	"log"
	"net/url"
)

const (
	LOG_PREFIX = "***** "
	LOG_SUFFIX = " *****"
)

func Print(text string, data ...interface{}) {
	return
	if len(text) > 0 {
		log.Println(LOG_PREFIX + text + LOG_SUFFIX)
	}
	if len(data) > 0 && data[0] != nil {
		log.Println(data...)
	}
}

func IsError(err error) bool {
	if err != nil {
		log.Println(err.Error())
		return true
	}
	return false
}

// Creates a query string prepended with ?. Usage: "/edit" + QueryString("hello")
func CreateQueryString(desc string) string {
	params := url.Values{
		"desc": {desc},
	}
	return "?" + params.Encode()
}
