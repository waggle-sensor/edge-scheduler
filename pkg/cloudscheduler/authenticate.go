package cloudscheduler

import (
	"fmt"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

type Authenticator interface {
	Authenticate(string) (*User, error)
	UpdatePermissionTableForUser(*User) error
}

func NewAuthenticator(authServerURL string, authToken string) Authenticator {
	// If no auth server is given we will use a fake auth granting any request
	if authServerURL == "" {
		return &FakeAuthenticator{}
	} else {
		return &RealAuthenticator{
			AuthServerURL: authServerURL,
			AuthToken:     authToken,
		}
	}
}

type FakeAuthenticator struct {
}

func (auth *FakeAuthenticator) Authenticate(token string) (*User, error) {
	return &User{
		Token: "sampletoken",
		Auth: &UserAuth{
			UserName: "superuser",
		},
	}, nil
}

func (auth *FakeAuthenticator) UpdatePermissionTableForUser(u *User) error {
	// TODO: We will need to think about how we fill the permission table for a fake user
	return nil
}

type RealAuthenticator struct {
	AuthServerURL string
	AuthToken     string
}

func (auth *RealAuthenticator) Authenticate(token string) (*User, error) {
	user := User{}
	user.Token = token
	// Get user name from auth service
	if err := auth.validateTokenAndGetAuth(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// validateTokenAndGetAuth attempts to validate the user with given token using
// the auth server. Upon validated the user information will be updated in `u` parameter.
func (auth *RealAuthenticator) validateTokenAndGetAuth(u *User) error {
	subPathString := "/users/~self"
	req := interfacing.NewHTTPRequest(auth.AuthServerURL)
	additionalHeader := map[string]string{
		"Authorization": fmt.Sprintf("Sage %s", u.Token),
		"Accept":        "application/json",
	}
	resp, err := req.RequestGet(subPathString, nil, additionalHeader)
	if err != nil {
		return err
	}
	decoder, err := req.ParseJSONHTTPResponse(resp)
	if err != nil {
		return err
	}
	var a UserAuth
	err = decoder.Decode(&a)
	if err != nil {
		return err
	}
	u.Auth = &a
	return nil
}

// UpdatePermissinoTableForUser uses the schedulers token to update user access of given user
func (auth *RealAuthenticator) UpdatePermissionTableForUser(u *User) error {
	subPathStringRegex := fmt.Sprintf("/users/%s/access", u.Auth.UserName)
	req := interfacing.NewHTTPRequest(auth.AuthServerURL)
	additionalHeader := map[string]string{
		"Authorization": fmt.Sprintf("Sage %s", auth.AuthToken),
		"Accept":        "application/json",
	}
	resp, err := req.RequestGet(subPathStringRegex, nil, additionalHeader)
	if err != nil {
		return err
	}
	decoder, err := req.ParseJSONHTTPResponse(resp)
	if err != nil {
		return err
	}
	type NodePermission struct {
		Vsn    string   `json:"vsn"`
		Access []string `json:"access"`
	}
	var permissions []NodePermission
	err = decoder.Decode(&permissions)
	if err != nil {
		return fmt.Errorf("Failed to decode permission table: %s", err.Error())
	}
	u.NodePermission = &UserPermissionTable{
		table: map[string]bool{},
	}
	for _, p := range permissions {
		for _, a := range p.Access {
			if a == PERMISSION_SCHEDULE {
				u.NodePermission.table[p.Vsn] = true
				break
			} else {
				u.NodePermission.table[p.Vsn] = false
			}
		}
	}
	u.NodePermission.lastUpdated = time.Now()
	return nil
}

type User struct {
	Token          string
	Auth           *UserAuth
	NodePermission *UserPermissionTable
}

func (u *User) GetUserName() string {
	if u.Auth != nil {
		return u.Auth.UserName
	}
	return ""
}

func (u *User) CanScheduleOnNode(nodeName string) (bool, error) {
	// Username must be provided
	if u.GetUserName() == "" {
		return false, fmt.Errorf("Username does not exist")
	}
	if u.NodePermission == nil {
		return false, fmt.Errorf("User permission table has not been updated")
	}
	result := u.NodePermission.HasSchedulePermission(nodeName)
	return result, nil
}

type UserAuth struct {
	Url           string `json:"url"`
	UserName      string `json:"username"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	IsStaff       bool   `json:"is_staff"`
	IsSuperUser   bool   `json:"is_superuser"`
	SshPublicKeys string `json:"ssh_public_keys"`
}

const (
	PERMISSION_SCHEDULE = "schedule"
)

type UserPermissionTable struct {
	table       map[string]bool
	lastUpdated time.Time
}

func (upt *UserPermissionTable) HasSchedulePermission(nodeName string) bool {
	if record, nodeExist := upt.table[nodeName]; nodeExist {
		return record
	}
	return false
}
