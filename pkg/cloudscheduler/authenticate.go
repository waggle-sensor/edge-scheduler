package cloudscheduler

type Authenticator interface {
	Authenticate(string) (bool, error)
}

func NewAuthenticator(authServerURL string) Authenticator {
	// If no auth server is given we will use a fake auth granting any request
	if authServerURL == "" {
		return &FakeAuthenticator{}
	} else {
		return &RealAuthenticator{
			AuthServerURL: authServerURL,
		}
	}
}

type FakeAuthenticator struct {
}

func (auth *FakeAuthenticator) Authenticate(token string) (bool, error) {
	return true, nil
}

type RealAuthenticator struct {
	AuthServerURL string
}

func (auth *RealAuthenticator) Authenticate(token string) (bool, error) {
	// req := interfacing.NewHTTPRequest(auth.AuthServerURL)
	// req.
	return true, nil
}
