package interfacing

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"gopkg.in/cenkalti/backoff.v1"
)

type HTTPRequest struct {
	BaseURL string
	c       *http.Client
}

func NewHTTPRequest(baseURL string) *HTTPRequest {
	return &HTTPRequest{
		BaseURL: baseURL,
		c:       &http.Client{},
	}
}

func (r *HTTPRequest) RequestGet(subPath string, queries url.Values, header map[string]string) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url = url.JoinPath(url.Path, subPath)
	if queries != nil {
		url.RawQuery = queries.Encode()
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	return r.c.Do(req)
}

func (r *HTTPRequest) RequestPost(subPath string, body []byte, header map[string]string) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url = url.JoinPath(url.Path, subPath)
	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	return r.c.Do(req)
}

func (r *HTTPRequest) RequestPostFromFile(subPath string, filePath string, queries url.Values, header map[string]string) (*http.Response, error) {
	url, err := url.Parse(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse %q: %s", r.BaseURL, err.Error())
	}
	url = url.JoinPath(url.Path, subPath)
	if queries != nil {
		url.RawQuery = queries.Encode()
	}
	blob, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(blob))
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(filePath)
	switch ext {
	case ".json":
		req.Header.Set("Content-Type", "application/json")
	case ".yaml", ".yml":
		req.Header.Set("Content-Type", "application/yaml")
	default:
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	return r.c.Do(req)
}

func (r *HTTPRequest) ParseJSONHTTPResponse(resp *http.Response) (decoder *json.Decoder, err error) {
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
	return json.NewDecoder(bytes.NewReader(stream)), nil
	// err = json.Unmarshal(stream, &body)
	// if err != nil {
	// 	return nil, fmt.Errorf("Failed to decode JSON body: %s", err.Error())
	// }
	// // body["StatusCode"] = resp.StatusCode
	// return body, nil
}

func (r *HTTPRequest) Subscribe(streamPath string, ch chan *datatype.Event, keepRetry bool) error {
	operation := func() error {
		resp, err := r.RequestGet(streamPath, nil, nil)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("could not connect to stream: %s", http.StatusText(resp.StatusCode))
		}
		defer resp.Body.Close()
		// patternEvent := regexp.MustCompile(`event:(.*?)\n`)
		// patternData := regexp.MustCompile(`data:(.*?)\n`)
		reader := bufio.NewScanner(resp.Body)
		reader.Split(ScanEvent)
		// var e *datatype.Event
		for {
			if reader.Scan() {
				line := reader.Text()
				logger.Debug.Printf("stream received: %s", line)
				eStart := strings.Index(line, "event:")
				eEnd := strings.Index(line, "data:")
				e := line[eStart+6 : eEnd]
				e = strings.Trim(e, " ")
				d := line[eEnd+5:]
				event := datatype.NewEventBuilder(datatype.EventType(e)).AddEntry("goals", d).Build()
				ch <- &event
				// if match := patternEvent.FindStringSubmatch(line); len(match) > 0 {
				// 	// if e != nil {
				// 	// 	fmt.Println("something is wrong")
				// 	// }
				// 	fmt.Printf("%s", match[1])
				// }
				// if match := patternData.FindStringSubmatch(line); len(match) > 0 {
				// 	// if e != nil {
				// 	// 	fmt.Println("something is wrong")
				// 	// }
				// 	fmt.Printf("%s", match[1])
				// }
			}
			if err := reader.Err(); err != nil {
				// reader.Scan returned false indicating an error
				return fmt.Errorf("Streaming encountered EOF or an error and considered as closed")
			}
			// event := reader.Text()
			// fmt.Printf("event received: %s", event)
			// for _, line := range strings.Split(event, "\n") {
			// 	if strings.
			// }
			// line, _ := reader.ReadSlice("\n\n")
			// fmt.Printf("event received: %s", line)
		}
	}
	go func() {
		for {
			err := backoff.Retry(operation, backoff.NewExponentialBackOff())
			logger.Error.Printf("Failed to subscribe %q: %s", streamPath, err.Error())
			if !keepRetry {
				logger.Info.Printf("keepRetry is false. Closing...")
				break
			}
			time.Sleep(5 * time.Second)
			logger.Info.Printf("Retrying to connect to %q in 5 seconds...", streamPath)
		}

	}()
	return nil
}

func ScanEvent(data []byte, atEOF bool) (advance int, token []byte, err error) {
	patEols := regexp.MustCompile(`[\r\n]+`)
	pat2Eols := regexp.MustCompile(`[\r\n]{2}`)
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if loc := pat2Eols.FindIndex(data); loc != nil && loc[0] >= 0 {
		// Replace newlines within string with a space
		s := patEols.ReplaceAll(data[0:loc[0]+1], []byte(" "))
		// Trim spaces and newlines from string
		s = bytes.Trim(s, "\n ")
		return loc[1], s, nil
	}

	if atEOF {
		// Replace newlines within string with a space
		s := patEols.ReplaceAll(data, []byte(" "))
		// Trim spaces and newlines from string
		s = bytes.Trim(s, "\r\n ")
		return len(data), s, nil
	}

	// Request more data.
	return 0, nil, nil
}
