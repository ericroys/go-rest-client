package rest

import (
	b64 "encoding/base64"
	"fmt"
	"net/http"
)

// Authenticator is interface for the adding
// appropriate authentication information to an
// *http.Request
type Authenticator interface {
	SetAuth(req *http.Request)
}

// AuthNoAuth is an Authenticator the provides
// no authentication for an *http.Request
type AuthNoAuth struct {
}

// SetAuth implements a do nothing implementation of
// Authenticator.SetAuth()
func (na AuthNoAuth) SetAuth(req *http.Request) {
	//remove any auth header
	req.Header.Del("Authorization")
}

// AuthBasic is an Authenticator for adding Basic
// authentication to an *http.Request
type AuthBasic struct {
	user string
	pass string
}

// create new AuthBasic
func NewAuthBasic(user string, password string) *AuthBasic {
	return &AuthBasic{user, password}
}

func (b *AuthBasic) getToken() string {
	s := fmt.Sprintf("%s:%s", b.user, b.pass)
	sEnc := b64.StdEncoding.EncodeToString([]byte(s))
	return sEnc
}

// SetAuth sets Basic Authentication info to an *http.Request
func (b AuthBasic) SetAuth(req *http.Request) {
	t := b.getToken()
	req.Header.Set("Authorization", "Basic "+t)
}

// AuthBearer is an Authenticator for adding Bearer
// authentication to an *http.Request
type AuthBearer struct {
	token string
}

// SetAuth sets Bearer token authentication for an *http.Request
func (b AuthBearer) SetAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+b.token)
}
