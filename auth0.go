package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type Auth0Config struct {
	ClientID     string
	ClientSecret string
	Domain       string
}

type Auth0Client struct {
	config      Auth0Config
	accessToken string
}

func NewAuth0Client(cfg Auth0Config) (Auth0Client, error) {
	client := Auth0Client{config: cfg}
	tok, err := client.getAccessToken()
	if err != nil {
		return Auth0Client{}, errors.Wrap(err, "initializing Auth0 client")
	}
	client.accessToken = tok
	return client, nil
}

// getAccessToken performs a client credentials OAuth2 flow against Auth0
// in order to allow this application to access the Auth0 management API.
// This is used during initialization to return a fully-configured Auth0
// client.
func (cl Auth0Client) getAccessToken() (string, error) {
	body := struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Audience     string `json:"audience"`
		GrantType    string `json:"grant_type"`
	}{
		ClientID:     cl.config.ClientID,
		ClientSecret: cl.config.ClientSecret,
		Audience:     fmt.Sprintf("https://%s/api/v2/", cl.config.Domain),
		GrantType:    "client_credentials",
	}
	bodyStr, err := json.Marshal(body)
	if err != nil {
		return "", errors.Wrap(err, "marshaling request body")
	}
	resp, err := http.Post(
		fmt.Sprintf("https://%s/oauth/token", cl.config.Domain),
		"application/json",
		bytes.NewBuffer(bodyStr))
	if err != nil {
		return "", errors.Wrap(err, "authorizing app for Auth0 API")
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("non-OK response from Auth0 when getting access token: %s", resp.Status)
	}

	var response struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", errors.Wrap(err, "decoding response from Auth0")
	}

	return response.AccessToken, nil
}

// GetUserDetails returns user details from the Auth0 management API. The
// caller must have already gotten an access token using GetAuth0AccessToken.
func (cl Auth0Client) GetUserDetails(userID string) (UserDetails, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v2/users/%s", cl.config.Domain, userID), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cl.accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UserDetails{}, errors.Wrap(err, "getting user details")
	}
	if resp.StatusCode != http.StatusOK {
		return UserDetails{}, errors.Errorf(
			"non-OK response from Auth0 when getting user details for %s: %s",
			userID, resp.Status)
	}

	var ud UserDetails
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&ud); err != nil {
		return UserDetails{}, errors.Wrapf(
			err,
			"decoding user details response from Auth0 for user %s",
			userID)
	}
	return ud, nil
}
