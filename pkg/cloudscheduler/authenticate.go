package cloudscheduler

import (
	"fmt"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

type Authenticator interface {
	Authenticate(string) (*User, error)
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

type RealAuthenticator struct {
	AuthServerURL string
	AuthPassword  string
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

type User struct {
	Token          string
	Auth           *UserAuth
	nodePermission *UserPermissionTable
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
	err := u.updateNodePermission()
	if err != nil {
		return false, fmt.Errorf("Failed to get node permission:%s", err.Error())
	}
	result := u.nodePermission.HasSchedulePermission(nodeName)
	return result, nil
}

func (u *User) updateNodePermission() error {
	if u.nodePermission == nil {
		u.nodePermission = &UserPermissionTable{
			table: map[string]bool{},
		}
	}
	// Once the table has updated, we do not update it again
	if time.Now().Sub(u.nodePermission.lastUpdated) > time.Duration(1*time.Minute) {
		return u.nodePermission.UpdateTable(u.GetUserName())
	}
	return nil
}

type UserAuth struct {
	Active   bool
	Scope    string
	ClientID string
	UserName string
	Exp      int
}

const (
	PERMISSION_SCHEDULE = "schedule"
)

type UserPermissionTable struct {
	table       map[string]bool
	lastUpdated time.Time
}

func (upt *UserPermissionTable) UpdateTable(userName string) error {
	// TODO: We need to avoid updating the table too frequently.
	//       With the following if statement the table is updated at most once an hour
	permissionServerURL := "https://access.sagecontinuum.org"
	subPathStringRegex := fmt.Sprintf("/profiles/%s/access", userName)
	req := interfacing.NewHTTPRequest(permissionServerURL)
	additionalHeader := map[string]string{
		"Accept": "application/json",
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
	for _, p := range permissions {
		for _, a := range p.Access {
			if a == PERMISSION_SCHEDULE {
				upt.table[p.Vsn] = true
				break
			} else {
				upt.table[p.Vsn] = false
			}
		}
	}
	upt.lastUpdated = time.Now()
	return nil
}

func (upt *UserPermissionTable) HasSchedulePermission(nodeName string) bool {
	if record, nodeExist := upt.table[nodeName]; nodeExist {
		return record
	}
	return false
}
