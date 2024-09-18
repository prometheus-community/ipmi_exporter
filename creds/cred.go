package creds

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type credentials struct {
	User string `json:"value1"`
	Pass string `json:"value2"`
}

const baseURL string = "localhost:8200"

func GetCreds(param string) (*credentials, error) {

	escapedParam := url.QueryEscape(param)
	url := fmt.Sprintf("%s/get-data?value=%s", baseURL, escapedParam)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var data credentials
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &data, nil
}
