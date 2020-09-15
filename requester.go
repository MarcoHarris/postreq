package requester

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/sethgrid/pester"
)

func NewService(concurrency int64, maxRetries int64) *Service {
	client := pester.New()
	client.Concurrency = 3
	client.MaxRetries = 5
	client.Backoff = pester.ExponentialBackoff
	client.KeepLog = true
	return &Service{
		Client: client,
	}
}

func (c *Service) Test(input string, params map[string]interface{}) ([]byte, int, error) {
	item := Item{}
	err := json.Unmarshal([]byte(input), &item)
	if err != nil {
		return nil, 999, err
	}

	req, err := generateRequest(item, params)
	if err != nil {
		log.Printf("Failed to form HTTP Request. Err: %v", err)
		return nil, 999, err
	}

	log.Printf("HTTP %v request to %v", req.Method, req.URL)
	res, reqErr := c.Client.Do(req)
	if reqErr != nil {
		log.Printf("Failed to do the request. Err: %v", reqErr)
		return nil, 999, err
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, 999, readErr
	}

	return body, res.StatusCode, nil
}

func generateRequest(item Item, params map[string]interface{}) (*http.Request, error) {

	endpoint := generateEndpoint(item, params)

	req, err := http.NewRequest(item.Request.Method, endpoint, bytes.NewBuffer(nil))
	if err != nil {
		log.Printf("Failed to form HTTP Request. Err: %v", err)
		return nil, err
	}

	q := req.URL.Query()
	for _, each := range item.Request.URL.Query {
		q.Set(each.Key, replacePlaceholder(each.Value, params))
	}
	req.URL.RawQuery = q.Encode()

	for _, each := range item.Request.Header {
		req.Header.Set(each.Key, replacePlaceholder(each.Value, params))
	}

	authValue, isAvailable := generateAuth(item, params)

	if isAvailable {
		req.Header.Set("Authorization", authValue)
	}
	return req, nil
}

func generateAuth(item Item, params map[string]interface{}) (string, bool) {
	var output string

	switch authType := item.Request.Auth.Type; authType {
	case "basic":
		value := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", params["username"], params["password"])))
		output = fmt.Sprintf("Basic %v", value)
		return output, true

	case "bearer":
		output = fmt.Sprintf("Bearer %v", params["AccessToken"])
		return output, true
	}

	return output, false
}

func generateEndpoint(item Item, params map[string]interface{}) string {
	var output []string

	for _, value := range item.Request.URL.Host {
		output = append(output, replacePlaceholder(value, params))
	}

	for _, value := range item.Request.URL.Path {
		output = append(output, replacePlaceholder(value, params))
	}

	return strings.Join(output, "/")
}

func replacePlaceholder(placeholder string, params map[string]interface{}) string {
	for key, value := range params {
		placeholder = strings.Replace(placeholder, fmt.Sprintf("{{%v}}", key), value.(string), -1)
		placeholder = strings.Replace(placeholder, fmt.Sprintf(":%v", key), value.(string), -1)
	}

	return placeholder
}
