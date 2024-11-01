package rest

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

/*
HTTPMethod provides for the type of method to use when making an http request
*/
type HTTPMethod int

const (
	//GET an http GET
	GET = iota
	//DELETE an http DELETE
	DELETE
	//PATCH an http PATCH
	PATCH
	//POST an http POST
	POST
	//PUT an http PUT
	PUT
)

func (d HTTPMethod) String() string {
	return [...]string{"GET", "DELETE", "PATCH", "POST", "PUT"}[d]
}

/*
ErrorHandler provides error handling based on code and
data context for a response. Based on code (http status code)
and data (http response message body), the ErrorHandler
will determine if there is an error, and return the error if
there is one, otherwise nil if no error found
*/
type ErrorHandler interface {
	Handle(code int, data []byte) error
}

/*
DefaultErrorHandler provides basic response error handling
using the status code only, and will be used if no other
more specific ErrorHandler is provided.
*/
type DefaultErrorHandler struct{}

/*
Handle checks if there is a valid status code and returns
nil if the code is good otherwise an error
*/
func (d DefaultErrorHandler) Handle(code int, data []byte) error {
	//basically only check only validate the status code for 200/201
	if code == 200 || code == 201 {
		return nil
	}
	return fmt.Errorf("error %d status code from request", code)
}

// stuct used for building a Requestable object
type requestInit struct {
	headers     map[string]string
	method      HTTPMethod
	auth        Authenticator
	contentType string
	handler     ErrorHandler
	msg         []byte
	url         string
	c           *http.Client
	r           *http.Request
}

/*
RequestableBuilder is an object used for purposes of
constructing a Requestable object. A Requestable object
should only ever be created using the builder.
*/
type RequestableBuilder struct {
	init requestInit
}

/*
Request is something that can be sent via rest. All
rest calls are made using it.
*/
type Request struct {
	client  *http.Client
	req     *http.Request
	handler ErrorHandler
}

/*
Send sends an http rest call, returning the response
as a byte array. An error is returned if there were
an issues with the request
*/
func (r *Request) Send() ([]byte, error) {

	log.Printf("Rest send [%s]", r.req.URL)
	resp, err := r.client.Do(r.req)
	if err != nil {
		return nil, err
	}

	//get the response stuff
	code, data, err := parseResponse(resp)
	//log.Printf("code %d\ndata: %s", code, string(data))
	//check for errors in response
	if err != nil {
		return nil, err
	}
	//deeper check for errors with provided error handler
	if err = r.handler.Handle(code, data); err != nil {
		//log.Printf("Http Send: code [%d], data [%s], Error [%v]", code, string(data), err)
		return nil, err
	}
	//return the bytes
	return data, nil
}

/*
NewRequestBuilder initializes a RequestableBuilder with required parameters.
Additional parameters can be supplied to the builder via its methods.
*/
func NewRequestBuilder(url string, client *http.Client) *RequestableBuilder {
	return &RequestableBuilder{
		init: requestInit{
			url:         url,
			c:           client,
			contentType: "application/json",
			method:      GET,
		},
	}
}

/*
Build builds a fully initialized Request object and validates
that all parameters are valid
*/
func (b *RequestableBuilder) Build() (*Request, error) {

	if err := b.validate(); err != nil {
		return nil, err
	}

	//set default handler if none
	if b.init.handler == nil {
		b.init.handler = DefaultErrorHandler{}
	}

	//generate bare http request
	r, err := http.NewRequest(b.init.method.String(), b.init.url, bytes.NewBuffer(b.init.msg))
	if err != nil {
		return nil, err
	}

	//set authentication (default to no auth if none provided)
	if b.init.auth == nil {
		var auth AuthNoAuth
		auth.SetAuth(r)
	} else {
		b.init.auth.SetAuth(r)
	}
	b.init.r = r

	//add content type
	b.init.r.Header.Set("Content-Type", b.init.contentType)

	//add headers
	if b.init.headers != nil {
		for k, v := range b.init.headers {
			if k != "" && v != "" {
				r.Header.Set(k, v)
			}
		}
	}
	return &Request{
		client:  b.init.c,
		req:     b.init.r,
		handler: b.init.handler,
	}, nil
}

/*
Header adds a header key and value pair. This method may be called as many
times as necessary to build a complete header list

	builder := NewRequestBuilder("myurl", client).
	           Header("mykey", "myvalue").
	           Header("Accepts", "application/json")
	           Header("host", "myhost")
*/
func (b *RequestableBuilder) Header(key, value string) *RequestableBuilder {
	if b.init.headers == nil {
		b.init.headers = make(map[string]string)
	}
	if len(key) > 0 {
		b.init.headers[key] = value
	}
	return b
}

/*
Auth allows addition of an Authenticator to the request. It only needs to be
called once, or not at all if no authentication is required
*/
func (b *RequestableBuilder) Auth(auth Authenticator) *RequestableBuilder {
	b.init.auth = auth
	return b
}

// Method allows addition of an HTTPMethod to the request (Default is POST)
func (b *RequestableBuilder) Method(method HTTPMethod) *RequestableBuilder {
	b.init.method = method
	return b
}

/*
ContentType is a convenience method for adding content type to the request.
A content type may be added directly via Header() as well, but requires
you to provide the key as well as the value
*/
func (b *RequestableBuilder) ContentType(ctype string) *RequestableBuilder {
	b.init.contentType = ctype
	return b
}

/*
ErrorHandler sets an ErrorHandler to be used for the request. If none is provided
a very basic one is used for default
*/
func (b *RequestableBuilder) ErrorHandler(handler ErrorHandler) *RequestableBuilder {
	b.init.handler = handler
	return b
}

// Message sets a message to request that will be sent
func (b *RequestableBuilder) Message(msg []byte) *RequestableBuilder {
	b.init.msg = msg
	return b
}

// validates the request has all its pieces and parts
func (b *RequestableBuilder) validate() error {
	//check for client
	if b.init.c == nil {
		return fmt.Errorf("http client not set")
	}
	//check auth  -> defaults to no auth
	// if b.init.auth == nil {
	// 	return fmt.Errorf("auth is not set")
	// }
	//check url
	if len(b.init.url) == 0 {
		return fmt.Errorf("a URL is not set")
	}
	if err := isValidURL(b.init.url); err != nil {
		return err
	}
	if (b.init.method == POST || b.init.method == PUT) &&
		(b.init.msg == nil || len(b.init.msg) == 0) {
		return fmt.Errorf("an http [%s] request requires a message body", b.init.method)
	}

	return nil
}

// checks if url is valid, errors of not
func isValidURL(s string) error {
	_, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("InvalidURL [%s]", s)
	}
	return nil
}

// handleResponse pulls status code and body content from http response
// if no body returns zero byte array
func parseResponse(response *http.Response) (code int, data []byte, err error) {

	//pull out status code
	code = response.StatusCode

	//get response body if we have one and
	//defer the body closure
	if response.Body != nil {
		defer response.Body.Close()
		data, err = io.ReadAll(response.Body)
		if err != nil {
			return code, nil, fmt.Errorf("unable to read response body")
		}
		return code, data, nil
	}
	return code, nil, nil
}
