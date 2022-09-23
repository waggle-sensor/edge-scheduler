package cloudscheduler

import (
	"fmt"

	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type Authenticator interface {
	Authenticate(string) (bool, error)
}

func NewAuthenticator(authServerURL string, authPassword string) Authenticator {
	// If no auth server is given we will use a fake auth granting any request
	if authServerURL == "" {
		return &FakeAuthenticator{}
	} else {
		return &RealAuthenticator{
			AuthServerURL: authServerURL,
			AuthPassword:  authPassword,
		}
	}
}

// curl -X POST -H 'Accept: application/json; indent=4' -H 'Content-Type: application/x-www-form-urlencoded' -H "Authorization: Basic c2FnZS1hcGktc2VydmVyOlY6VUJFZnNEWkc=" -d 'token=CJ2Q20Y0F38L3TLGD92W'  http://192.168.42.146:80/token_info/

type User struct {
	Token string
	auth  *SageAuth
}

type SageAuth struct {
	Active   bool
	Scope    string
	ClientID string
	UserName string
	Exp      int
}

type FakeAuthenticator struct {
}

func (auth *FakeAuthenticator) Authenticate(token string) (bool, error) {
	return true, nil
}

type RealAuthenticator struct {
	AuthServerURL string
	AuthPassword  string
}

func (auth *RealAuthenticator) Authenticate(token string) (bool, error) {
	user := User{}
	user.Token = token
	// Get user name from auth service
	if err := auth.validateTokenAndGetAuth(&user); err != nil {
		return false, err
	}

	return true, nil
}

func (auth *RealAuthenticator) validateTokenAndGetAuth(u *User) error {
	subPathString := "/token_info/"
	payload := fmt.Sprintf("token=%s", u.Token)
	req := interfacing.NewHTTPRequest(auth.AuthServerURL)
	additionalHeader := map[string]string{
		"Authorization": fmt.Sprintf("Basic %s", auth.AuthPassword),
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
	}
	resp, err := req.RequestPost(subPathString, []byte(payload), additionalHeader)
	if err != nil {
		return err
	}
	body, err := req.ParseJSONHTTPResponse(resp)
	if err != nil {
		return err
	}
	logger.Debug.Println(body)
	return nil
}
