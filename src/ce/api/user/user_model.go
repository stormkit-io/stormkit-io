package user

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	null "gopkg.in/guregu/null.v3"
)

// Subscription represents a user subscription and its details.
type Subscription struct {
	// Name represents the subscription name, as in the nickname in
	// Stripe plans.
	Name string `json:"name"` // starter | medium | enterprise

	// Email represents the user email.
	Email string `json:"-"`

	// UserID represents the user id.
	UserID types.ID `json:"-"`
}

type UserMeta struct {
	StripeCustomerID string `json:"stripeCustomerId,omitempty"`
	SeatsPurchased   int    `json:"seats,omitempty"`   // Number of seats purchased
	PackageName      string `json:"package,omitempty"` // SubscriptionPackage
}

// Scan implements the sql.Scanner interface for UserMeta
func (um *UserMeta) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)

	if !ok {
		return fmt.Errorf("cannot scan %T into UserMeta", value)
	}

	return json.Unmarshal(bytes, um)
}

// Value implements the driver.Valuer interface for UserMeta
func (um UserMeta) Value() (driver.Value, error) {
	return json.Marshal(um)
}

// User represents a user object.
type User struct {
	ID          types.ID      `json:"id,string"`
	DisplayName string        `json:"displayName"`
	Avatar      null.String   `json:"avatar"`
	Emails      []oauth.Email `json:"-"`
	CreatedAt   utils.Unix    `json:"memberSince"`
	LastLogin   utils.Unix    `json:"-"`
	FirstName   null.String   `json:"-"`
	LastName    null.String   `json:"-"`
	IsAdmin     bool          `json:"isAdmin,omitempty"`
	IsNew       bool          `json:"-"` // Whether the user is newly created or not.
	IsApproved  null.Bool     `json:"-"` // Null => pending, FALSE => not allowed, TRUE => allowed
	Metadata    UserMeta      `json:"metadata,omitempty"`
}

type Mail struct {
	Endpoint string `json:"-"`
	Email    string `json:"email"`
}

// New creates and returns a new user instance.
func New(email string) *User {
	return &User{
		Emails: []oauth.Email{
			{
				Address:    strings.ToLower(strings.TrimSpace(email)),
				IsPrimary:  true,
				IsVerified: true,
			},
		},
	}
}

// PrimaryEmail returns the primary the email of the user. If none is found,
// it returns the first email. Otherwise returns an empty string.
func (u User) PrimaryEmail() string {
	for _, email := range u.Emails {
		if email.IsPrimary {
			return email.Address
		}
	}

	if len(u.Emails) > 0 {
		return u.Emails[0].Address
	}

	return ""
}

// HasEmail checks if the user has the given email or not.
func (u User) HasEmail(email string) bool {
	for _, e := range u.Emails {
		if strings.EqualFold(e.Address, email) {
			return true
		}
	}

	return false
}

// FullName returns the first name and last name of the user.
func (u User) FullName() string {
	firstName := u.FirstName.ValueOrZero()
	lastName := u.LastName.ValueOrZero()

	if firstName != "" && lastName != "" {
		return fmt.Sprintf("%s %s", firstName, lastName)
	}

	if firstName != "" {
		return firstName
	}

	return lastName
}

// Display returns a name to display. It first relies on FirstName + LastName.
// If none is provided, it will check the display name.
// Otherwise the primary email address.
func (u User) Display() string {
	if u.FirstName.ValueOrZero() != "" || u.LastName.ValueOrZero() != "" {
		return u.FullName()
	}

	if u.DisplayName != "" {
		return u.DisplayName
	}

	return u.PrimaryEmail()
}

// JSON casts the user object into a map, ready to be sent to the frontend.
func (u User) JSON() map[string]any {
	userData := map[string]any{
		"id":          u.ID.String(),
		"avatar":      u.Avatar,
		"email":       u.PrimaryEmail(),
		"fullName":    u.FullName(),
		"displayName": u.DisplayName,
		"memberSince": u.CreatedAt.Unix(),
		"package": map[string]any{
			"id":    utils.GetString(u.Metadata.PackageName, config.PackageFree),
			"seats": utils.GetInt(u.Metadata.SeatsPurchased, 1),
		},
	}

	if u.IsAdmin {
		userData["isAdmin"] = true
	}

	return userData
}

// IsAuthorizedToLogin checks if the user is authorized to login based on
// the current sign-up mode and the user's approval status.
func (u User) IsAuthorizedToLogin() bool {
	// If user was rejected, deny login immediately.
	if u.IsApproved.Valid && !u.IsApproved.ValueOrZero() {
		return false
	}

	if u.IsApproved.Valid && u.IsApproved.ValueOrZero() {
		return true
	}

	cnf := admin.MustConfig()

	for _, email := range u.Emails {
		if cnf.IsUserWhitelisted(email.Address) {
			return true
		}
	}

	return false
}

// ConnectedAccount represents a connected account.
type ConnectedAccount struct {
	Provider               string `json:"provider"`
	URL                    string `json:"url"`
	DisplayName            string `json:"displayName"`
	HasPersonalAccessToken bool   `json:"hasPersonalAccessToken"`
}
