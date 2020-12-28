package utils

import "example.com/kendrick/api"

var (
	TEST_REQUEST = api.Request{
		Id:   "123456",
		Type: "TEST",
		Data: make(map[string]string, 3),
	}
	TEST_RESPONSE = api.Response{
		Id:          "123456",
		Code:        api.CREDENTIALS_VALID,
		Description: "Description",
		Data:        make(map[string]string, 3),
	}
	TEST_REPETITIONS = 10
)
