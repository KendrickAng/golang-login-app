package utils

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"example.com/kendrick/api"
	log "github.com/sirupsen/logrus"
	"testing"
)

func BenchmarkGobEncodeDecode(b *testing.B) {
	// Buffer is a variable-sized buffer
	var network bytes.Buffer
	var payload api.Request
	var response api.Response
	enc := gob.NewEncoder(&network)
	dec := gob.NewDecoder(&network)
	for i := 0; i < b.N; i++ {
		for i := 0; i < TEST_REPETITIONS; i++ {
			err := enc.Encode(TEST_REQUEST)
			if err != nil {
				log.Fatalln(err)
			}
			err = enc.Encode(TEST_RESPONSE)
			if err != nil {
				log.Fatalln(err)
			}
		}
		for i := 0; i < TEST_REPETITIONS; i++ {
			err := dec.Decode(&payload)
			if err != nil {
				log.Fatalln(err)
			}
			err = dec.Decode(&response)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func BenchmarkJsonEncodeDecode(b *testing.B) {
	var network bytes.Buffer
	var payload api.Request
	var response api.Response
	enc := json.NewEncoder(&network)
	dec := json.NewDecoder(&network)
	for i := 0; i < b.N; i++ {
		for i := 0; i < TEST_REPETITIONS; i++ {
			err := enc.Encode(TEST_REQUEST)
			if err != nil {
				log.Fatalln(err)
			}
			err = enc.Encode(TEST_RESPONSE)
			if err != nil {
				log.Fatalln(err)
			}
		}
		for i := 0; i < TEST_REPETITIONS; i++ {
			err := dec.Decode(&payload)
			if err != nil {
				log.Fatalln(err)
			}
			err = dec.Decode(&response)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}
