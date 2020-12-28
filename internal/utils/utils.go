package utils

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

func IsError(err error) bool {
	if err != nil {
		log.Error(err)
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
