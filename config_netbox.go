package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/prometheus/common/log"
)

var (
	cache          netboxCache
	cacheTTL       = 8 * time.Hour
	cacheTTLJitter = func() time.Duration { return time.Duration(rand.Intn(120)) * time.Minute }
)

type netboxHost struct {
	Name         string            `json:"name"`
	CustomFields map[string]string `json:"custom_fields"`
}

type netboxSecret struct {
	Name      string           `json:"name"`
	Plaintext string           `json:"plaintext"`
	Role      netboxSecretRole `json:"role"`
}

type netboxSecretRole struct {
	Slug string `json:"slug"`
}

type netboxGetHostsResult struct {
	Results []netboxHost `json:"results"`
}

type netboxGetSecretsResult struct {
	Results []netboxSecret `json:"results"`
}

type netboxCachedHost struct {
	address  string
	username string
	password string
	sync     time.Time
}

type netboxCache struct {
	hosts sync.Map
}

func (s *IPMIConfig) updateFrom(cached *netboxCachedHost) {
	if cached.address != "" {
		s.Address = cached.address
	}
	if cached.username != "" {
		s.User = cached.username
	}
	if cached.password != "" {
		s.Password = cached.password
	}
}

// RetrieveFromNetBox retrieves info from NetBox or cache
func (s *IPMIConfig) RetrieveFromNetBox(target string) {
	if iff, ok := cache.hosts.Load(target); ok {
		if cached, ok := iff.(*netboxCachedHost); ok {
			if time.Now().Add(-cacheTTL - cacheTTLJitter()).Before(cached.sync) {
				s.updateFrom(cached)
				return
			}
			cache.hosts.Delete(target)
		}
	}

	address, username, password := s.RawRetrieveFromNetBox(target)
	if address+username+password == "" {
		return
	}

	// Update cache
	cached := &netboxCachedHost{address, username, password, time.Now()}
	cache.hosts.LoadOrStore(target, cached)

	s.updateFrom(cached)
}

// RawRetrieveFromNetBox retrieves IPMI URL, username and password from NetBox.
// Only updates fields if they are available on NetBox.
func (s *IPMIConfig) RawRetrieveFromNetBox(target string) (address, username, password string) {
	saveAddress := ""
	if s.NetBox.Address == "" {
		return
	}
	request, err := http.NewRequest("GET", s.NetBox.Address+"/api/dcim/devices/", nil)
	if err != nil {
		log.Error(err)
		return
	}
	request.Header.Add("authorization", fmt.Sprintf("Token %s", s.NetBox.Token))
	request.Header.Add("x-session-key", s.NetBox.SessionKey)
	q := request.URL.Query()
	for key, value := range s.NetBox.Params {
		q.Add(key, value)
	}
	q.Add("q", target)
	request.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Error(err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return
	}
	var hosts netboxGetHostsResult
	if err := json.Unmarshal(body, &hosts); err != nil || len(hosts.Results) == 0 {
		log.Error(err)
		return
	}
	host := hosts.Results[0]
	if s.NetBox.IPMICustomField != "" {
		if fromNetbox, ok := host.CustomFields[s.NetBox.IPMICustomField]; ok {
			ipmiURL, err := url.Parse(fromNetbox)
			if err != nil {
				log.Error(err)
			} else {
				saveAddress = ipmiURL.Hostname()
			}
		}
	}

	if s.NetBox.CredentialsSecret == "" {
		address = saveAddress
		return
	}
	request, err = http.NewRequest("GET", s.NetBox.Address+"/api/secrets/secrets/", nil)
	if err != nil {
		log.Error(err)
		return
	}
	request.Header.Add("authorization", fmt.Sprintf("Token %s", s.NetBox.Token))
	request.Header.Add("x-session-key", s.NetBox.SessionKey)
	q = request.URL.Query()
	q.Add("device", host.Name)
	request.URL.RawQuery = q.Encode()
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		log.Error(err)
		return
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return
	}
	var secrets netboxGetSecretsResult
	if err := json.Unmarshal(body, &secrets); err != nil {
		log.Error(err)
		return
	}
	for _, secret := range secrets.Results {
		if secret.Role.Slug == s.NetBox.CredentialsSecret {
			address = saveAddress
			username = secret.Name
			password = secret.Plaintext
			break
		}
	}
	return
}
