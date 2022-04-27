package interfacing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

type HTTPRequest struct {
	BaseURL string
}

func NewHTTPRequest(baseURL string) *HTTPRequest {
	return &HTTPRequest{
		BaseURL: baseURL,
	}
}

func (r *HTTPRequest) RequestGet(subPath string, queries url.Values) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url.Path = path.Join(url.Path, subPath)
	url.RawQuery = queries.Encode()
	return http.Get(url.String())
}

func (r *HTTPRequest) RequestPost(subPath string, body []byte) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url.Path = path.Join(url.Path, subPath)
	return http.Post(url.String(), "application/json", bytes.NewBuffer(body))
}

func (r *HTTPRequest) RequestPostFromFile(subPath string, filePath string) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url.Path = path.Join(url.Path, subPath)
	blob, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return http.Post(url.String(), "application/json", bytes.NewBuffer(blob))
}

func (r *HTTPRequest) RequestPostFromFileWithQueries(subPath string, filePath string, queries url.Values) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url.Path = path.Join(url.Path, subPath)
	url.RawQuery = queries.Encode()
	blob, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return http.Post(url.String(), "application/json", bytes.NewBuffer(blob))
}

func (r *HTTPRequest) ParseJSONHTTPResponse(resp *http.Response) (body map[string]interface{}, err error) {
	defer resp.Body.Close()
	stream, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body of response: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Returned %q: %s", resp.Status, string(stream))
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Content-Type is not JSON: %s", resp.Header.Get("Content-Type"))
	}
	err = json.Unmarshal(stream, &body)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode JSON body: %s", err.Error())
	}
	// body["StatusCode"] = resp.StatusCode
	return body, nil
}
