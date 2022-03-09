package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	Version     string
	debug       bool
	name        string
	node        string
	privileged  bool
	job         string
	selectorStr string
	followLog   bool
	entrypoint  string
	stdin       bool
	tty         bool
)

func init() {
	// To prevent printing the usage when commands end with an error
	// rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "flag to debug")
}

var rootCmd = &cobra.Command{
	Use: "sesctl [FLAGS] [COMMANDS]",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !debug {
			logger.Debug.SetOutput(io.Discard)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("SAGE edge scheduler client version: %s\n", Version)
		fmt.Printf("sesctl --help for more information\n")
	},
	// ValidArgs: []string{"deploy", "logs"},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func RequestHTTP(url string) (*http.Response, error) {
	return http.Get(url)
}

func ParseJSONHTTPResponse(resp *http.Response) (body map[string]interface{}, err error) {
	defer resp.Body.Close()
	stream, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body of response: %s", err.Error())
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Content-Type is not JSON: %s", resp.Header.Get("Content-Type"))
	}
	err = json.Unmarshal(stream, &body)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode JSON body: %s", err.Error())
	}
	body["StatusCode"] = resp.StatusCode
	return body, nil
}

func PrintJobsPretty(jobs []*datatype.Job) string {
	// formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, "NAME", maxLengthStatus+3, "STATUS", maxLengthStartTime+3, "START_TIME", maxLengthDuration+3, "RUNNING_TIME")
	// formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, plugin.Name, maxLengthStatus+3, status, maxLengthStartTime+3, startTime, maxLengthDuration+3, duration)
	return ""
}
