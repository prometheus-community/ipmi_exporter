package creds

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type credentials struct {
	User string `json:"username"`
	Pass string `json:"password"`
}

const baseURL string = "http://10.50.1.65:8200"

func GetCreds(param string) (*credentials, error) {
	escapedParam := url.QueryEscape(param)
	url := fmt.Sprintf("%s/get-data?target=%s", baseURL, escapedParam)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Use a map to unmarshal the response
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Access username and password from the map
	user, ok := data["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username not found or invalid type")
	}

	pass, ok := data["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password not found or invalid type")
	}

	return &credentials{User: user, Pass: pass}, nil
}
