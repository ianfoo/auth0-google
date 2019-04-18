package main

import (
	"fmt"
	"time"
)

// UserDetails represents the data that Auth0 returns from its /users
// management API.
type UserDetails struct {
	UserClaims
	Identities   []Identity `json:"identities"`
	CreatedAtStr string     `json:"created_at"`
	LastLoginStr string     `json:"last_login"`
	LastIP       string     `json:"last_ip"`
	LoginsCount  uint       `json:"logins_count"`
}

// Identity returns the identity record for the specified provider. If the
// provider does not exist in the UserDetails identities, an error is returned.
func (ud UserDetails) Identity(provider string) (Identity, error) {
	for _, id := range ud.Identities {
		if id.Provider == provider {
			return id, nil
		}
	}
	return Identity{}, fmt.Errorf("no identity for provider %q", provider)
}

// UserClaims represent OIDC User Claims, as returned by Auth0.
//
// See https://openid.net/specs/openid-connect-basic-1_0-28.html#StandardClaims
// Note that the spec indicates that UpdatedAt is seconds from epoch, but Auth0
// returns an RFC3339 string, hence the need for a method to get the udpated at
// date.
type UserClaims struct {
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	MiddleName    string `json:"middle_name"`
	FamilyName    string `json:"family_name"`
	Nickname      string `json:"nickname"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	UpdatedAtStr  string `json:"updated_at"`
}

// UpdatedAt gets a time.Time for the RFC3339 string returned by Auth0 in
// the updated_at field.
func (uc UserClaims) UpdatedAt() (time.Time, error) {
	return time.Parse(time.RFC3339, uc.UpdatedAtStr)
}

// Identity represents an Auth0 identity.
type Identity struct {
	UserID       string `json:"user_id"`
	Provider     string `json:"provider"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    uint   `json:"expires_in"`
	Connection   string `json:"connection"`
	IsSocial     bool   `json:"isSocial"`
}

// ToUserConfig returns an internal user config from an Auth0 identity.
//
// This is probably not really necessary, but the response from Auth0 and
// the way of persisting it locally were conceived of separately. A better
// representation should be used.
func (id Identity) ToUserConfig() UserConfig {
	return UserConfig{
		AccessToken:  id.AccessToken,
		ExpiresAt:    time.Now().Add(time.Duration(id.ExpiresIn) * time.Second),
		RefreshToken: id.RefreshToken,
	}
}
