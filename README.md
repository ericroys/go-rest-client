# Request wrapper for httpClient

## Description

A wrapper for the httpClient that simplifies making Restful calls by using a builder to create the request.

## Example Useage

```
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/ericroys/go-rest-client"
)

//setup a basic http client to use or customize as needed
func getClient() *http.Client {
	trans := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	return &http.Client{
		Transport: trans,
		Timeout:   2 * time.Second,
	}
}

func main() {

	//setup a message
	message := []byte(`{"text": "the quick brown fox"}`)

	//build the request
	r, err := rest.NewRequestBuilder("http://localhost.com", getClient()).
		Auth(rest.NewAuthBasic("bob", "needs_access")).
		Header("Accept", "application/json").
		Header("From", "bob.go@someaddress.email").
		Method(rest.POST).
		Message(message).
		Build()
	if err != nil {
		log.Fatalf(`oops! something went wrong. %s`, err.Error())
	}

	//send the request
	res, err := r.Send()
	if err != nil {
		log.Fatalf(`oops! something went wrong. %s`, err.Error())
	}
	/*unmarshall the response, which is returned as a byte array
	  so we can check individual return bits
	  &response is the struct having the transformation info
	  e.g
	  response struct {
		  Result map[string][]string  `json:"result"`
		  Count  int                  `json:"total"`
		  Error  string               `json:"error"`
		  Next   string               `json:"nextId"`
	  }
	*/
	//err = json.UnmarshalFromString((string(res[:])), &response)
}

```
