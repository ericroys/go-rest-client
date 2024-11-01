package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
)

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

var (
	server   *httptest.Server
	json     = jsoniter.ConfigCompatibleWithStandardLibrary
	fakeUri  = "http://localhost:8080"
	response struct {
		Headers map[string][]string `json:"headers"`
		Method  string              `json:"method"`
		Error   string              `json:"error"`
		Data    string              `json:"data"`
	}
)

func mockRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//define our reponse map
	m := make(map[string]interface{})
	//shove the headers in
	m["headers"] = r.Header
	// for k, v := range r.Header {
	// 	fmt.Printf(`key: %s  -> value: %s`, k, v)
	// }
	//shove in the method
	method := r.Method
	if method == "" {
		method = "GET"
	}
	m["method"] = r.Method
	//just auto sending an ok
	w.WriteHeader(http.StatusOK)
	//get body contents to return
	data, err := io.ReadAll(r.Body)
	//write in an error if there is one
	if err != nil {
		m["error"] = err.Error()
	} else {
		m["data"] = string(data[:])
	}
	json.NewEncoder(w).Encode(m)
}

/*
Got the request from building a Requestable builder
Throws an error if build fails
returns request if success
*/
func getRequest(builder *RequestableBuilder, t *testing.T) *Request {
	r, err := builder.Build()
	if err != nil {
		t.Error(err)
	}
	return r
}

/*
Run tests on the request builder itself
*/
func TestRequestableBuilder(t *testing.T) {

	testCases := []struct {
		input     *RequestableBuilder
		testName  string
		expectErr bool
	}{
		{
			input:     NewRequestBuilder(fakeUri, getClient()),
			testName:  "No error on base minimum (i.e. all defaults are set/used)",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder("", getClient()),
			testName:  "Error on missing bad url",
			expectErr: true,
		},
		{
			input:     NewRequestBuilder(fakeUri, nil),
			testName:  "Error on missing client",
			expectErr: true,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Auth(AuthBasic{"bob", "haspassword"}),
			testName:  "No Error on add Basic",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Auth(AuthBearer{token: "faketoken"}),
			testName:  "No Error on add Bearer",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Auth(AuthNoAuth{}),
			testName:  "No error on add no auth",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).ContentType("application/json"),
			testName:  "No error add content type",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).ContentType(""),
			testName:  "No error add content type empty (should be ignored)",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Header("mykey", "bob"),
			testName:  "No error add header",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Header("", ""),
			testName:  "No error add empty header (should be ignored)",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Message(nil),
			testName:  "No error adding nill message",
			expectErr: false,
		},
		{
			input:     NewRequestBuilder(fakeUri, getClient()).Message([]byte("test")),
			testName:  "No error add message",
			expectErr: false,
		},
		// { unnecessary since method can only be one of the valid values or defaults to GET
		// 	input:     NewRequestBuilder(fakeUri, getClient()).Method(),
		// 	testName:  "No error on set no method ?? ",
		// 	expectErr: false,
		// },
	}

	for k, v := range testCases {
		_ = k
		r, err := v.input.Build()
		_ = r
		if v.expectErr && err == nil {
			t.Errorf("failed {%s}, expected(%t) but received(%t)", v.testName,
				v.expectErr, err != nil)
		}
		if !v.expectErr && err != nil {
			t.Errorf("failed {%s}, expected(%t) but received(%t)", v.testName,
				v.expectErr, err == nil)
		}
	}
}

func TestRequestableRequestE2E(t *testing.T) {

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockRequestHandler(w, r)
	}))

	testCases := []struct {
		input           *Request
		testName        string
		expectedHeaders map[string]string
		expectedMethod  string
		expectedMessage string
		expectErr       bool
	}{
		{
			input:           getRequest(NewRequestBuilder(server.URL, getClient()).Auth(AuthBasic{"bob", "haspass"}), t),
			testName:        "Basic auth",
			expectedHeaders: map[string]string{"Authorization": "Basic " + NewAuthBasic("bob", "haspass").getToken()},
			expectedMethod:  "GET",
			expectedMessage: "",
			expectErr:       false,
		},
		{
			input:           getRequest(NewRequestBuilder(server.URL, getClient()).Auth(AuthBearer{"mytoken"}), t),
			testName:        "Token auth",
			expectedHeaders: map[string]string{"Authorization": "Bearer mytoken"},
			expectedMethod:  "GET",
			expectedMessage: "",
			expectErr:       false,
		},
		{
			input:           getRequest(NewRequestBuilder(server.URL, getClient()).Method(POST).Message([]byte(`{"key": "value"}`)), t),
			testName:        "Post expect success",
			expectedHeaders: map[string]string{},
			expectedMethod:  "POST",
			expectedMessage: `{"key": "value"}`,
			expectErr:       false,
		},
		{
			input:           getRequest(NewRequestBuilder(server.URL, getClient()).Header("Mykey", "myvalue").Header("Mykey1", "myvalue1"), t),
			testName:        "Added header Test",
			expectedHeaders: map[string]string{"Mykey": "myvalue", "Mykey1": "myvalue1"},
			expectedMethod:  "GET",
			expectedMessage: "",
			expectErr:       false,
		},
	}

	for _, v := range testCases {

		//send to test endpoint, check for error
		r, err := v.input.Send()
		if err != nil {
			t.Errorf("failed {%s}, expected(%t) but received(%t) with error( %s )", v.testName,
				v.expectErr, err != nil, err)
		}
		//with response run through additional tests

		//unmarshall the response so we can check individual return bits
		err = json.UnmarshalFromString((string(r[:])), &response)
		if err != nil {
			t.Errorf("failed {%s}, expected(%t) but received(%t) with error( %s )", v.testName,
				v.expectErr, err != nil, err)
		}
		//header tests
		if len(v.expectedHeaders) > 0 {
			hcheck := "header check"
			for kh, vh := range v.expectedHeaders {
				h := response.Headers[kh]
				if h == nil {
					t.Errorf("failed {%s-%s-key}, expected(%s) but received(%s)", v.testName,
						hcheck, kh, "nil")
				}
				tx := strings.Join(h, "")

				if tx != vh {
					t.Errorf("failed {%s-%s-value}, expected(%s) but received(%s) ", v.testName, hcheck,
						vh, tx)
				}
			}
		}
		//method test
		if v.expectedMethod != response.Method {
			t.Errorf("failed {%s}, expected(%s) but received(%s)", v.testName,
				v.expectedMethod, response.Method)
		}
		//message test
		if v.expectedMessage != response.Data {
			t.Errorf("failed {%s}, expected(%s) but received(%s)", v.testName,
				v.expectedMethod, response.Method)
		}
	}
}
