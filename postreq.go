package postreq

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

func NewService(concurrency int, maxRetries int) *Service {
	client := pester.New()
	client.Concurrency = concurrency
	client.MaxRetries = maxRetries
	client.Backoff = pester.ExponentialBackoff
	client.KeepLog = true
	return &Service{
		Client: client,
	}
}

func (c *Service) Do(input string, params map[string]interface{}) (http.Header, []byte, int, error) {
	item := Item{}
	err := json.Unmarshal([]byte(input), &item)
	if err != nil {
		return nil, nil, 999, err
	}

	req := generateRequest(item, params)

	log.Printf("HTTP %v request to %v", req.Method, req.URL)
	res, reqErr := c.Client.Do(req)
	if reqErr != nil {
		log.Printf("Failed to do the request. Err: %v", reqErr)
		return nil, nil, 999, reqErr
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, nil, 999, readErr
	}

	return res.Header, body, res.StatusCode, nil
}

func generateRequest(item Item, params map[string]interface{}) *http.Request {
	endpoint := generateEndpoint(item.Request.URL.Host, item.Request.URL.Path, params)

	req, err := http.NewRequest(item.Request.Method, endpoint, bytes.NewBuffer(nil))
	if err != nil {
		log.Printf("Failed to form HTTP Request. Err: %v", err)
	}

	q := req.URL.Query()
	for _, each := range item.Request.URL.Query {

		if newValue, ok := getValue(each.Value, params); ok {
			q.Set(each.Key, newValue)
		}
	}
	req.URL.RawQuery = q.Encode()

	for _, each := range item.Request.Header {
		if newValue, ok := getValue(each.Value, params); ok {
			req.Header.Set(each.Key, newValue)
		}

	}

	authValue, isAvailable := generateAuth(item.Request.Auth.Type, params)

	if isAvailable {
		req.Header.Set("Authorization", authValue)
	}
	return req
}

func generateAuth(inputAuthType string, params map[string]interface{}) (string, bool) {
	var output string
	var username, password, token string

	switch authType := inputAuthType; authType {
	case "basic":

		if _, ok := params["username"]; ok {
			username = params["username"].(string)
		}
		if _, ok := params["password"]; ok {
			password = params["password"].(string)
		}

		value := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", username, password)))
		output = fmt.Sprintf("Basic %v", value)
		return output, true

	case "bearer":

		if _, ok := params["accessToken"]; ok {
			token = params["accessToken"].(string)
		}

		output = fmt.Sprintf("Bearer %v", token)
		return output, true
	}

	return output, false
}

func generateEndpoint(hosts []string, paths []string, params map[string]interface{}) string {
	var output []string

	for _, value := range hosts {
		if newValue, ok := getValue(value, params); ok {
			output = append(output, newValue)
		}
	}

	for _, value := range paths {

		if newValue, ok := getValue(value, params); ok {
			output = append(output, newValue)
		}
	}

	return strings.Join(output, "/")
}

func getValue(value string, params map[string]interface{}) (string, bool) {
	if ok := isPlaceholder(value); !ok {
		return value, true
	}

	if result, replaced := replacePlaceholder(value, params); replaced {
		return result, true
	}

	return "", false
}

func isPlaceholder(value string) bool {

	return strings.ContainsAny(value, ":{")
}

func replacePlaceholder(placeholder string, params map[string]interface{}) (string, bool) {
	output := placeholder
	for key, value := range params {
		output = strings.Replace(output, fmt.Sprintf("{{%v}}", key), fmt.Sprintf("%v", value), -1)
		output = strings.Replace(output, fmt.Sprintf(":%v", key), fmt.Sprintf("%v", value), -1)
	}

	return output, output != placeholder
}
