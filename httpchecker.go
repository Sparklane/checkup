package checkup

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/drone/envsubst"
)

// HTTPChecker implements a Checker for HTTP endpoints.
type HTTPChecker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// URL is the URL of the endpoint.
	URL string `json:"endpoint_url"`

	// UpStatus is the HTTP status code expected by
	// a healthy endpoint. Default is http.StatusOK.
	UpStatus int `json:"up_status,omitempty"`

	// ThresholdRTT is the maximum round trip time to
	// allow for a healthy endpoint. If non-zero and a
	// request takes longer than ThresholdRTT, the
	// endpoint will be considered unhealthy. Note that
	// this duration includes any in-between network
	// latency.
	ThresholdRTT time.Duration `json:"threshold_rtt,omitempty"`

	// MustContain is a string that the response body
	// must contain in order to be considered up.
	// NOTE: If set, the entire response body will
	// be consumed, which has the potential of using
	// lots of memory and slowing down checks if the
	// response body is large.
	MustContain string `json:"must_contain,omitempty"`

	// MustNotContain is a string that the response
	// body must NOT contain in order to be considered
	// up. If both MustContain and MustNotContain are
	// set, they are and-ed together. NOTE: If set,
	// the entire response body will be consumed, which
	// has the potential of using lots of memory and
	// slowing down checks if the response body is large.
	MustNotContain string `json:"must_not_contain,omitempty"`

	// Attempts is how many requests the client will
	// make to the endpoint in a single check.
	Attempts int `json:"attempts,omitempty"`

	// AttemptSpacing spaces out each attempt in a check
	// by this duration to avoid hitting a remote too
	// quickly in succession. By default, no waiting
	// occurs between attempts.
	AttemptSpacing time.Duration `json:"attempt_spacing,omitempty"`

	// Retries is how many retry requests.
	Retries int `json:"retries,omitempty"`

	// RetrySpacing spaces out each retry in a check
	// by this duration to avoid hitting a remote too
	// quickly in succession. By default, no waiting
	// occurs between retries.
	RetrySpacing time.Duration `json:"retry_spacing,omitempty"`

	// IgnoreTimes times when down check result should be ignored
	// because of recurring maintenance for example
	IgnoreTimes []string `json:"ignore_times,omitempty"`

	// IgnoreDuration duration when down check result should be ignored
	// because of recurring maintenance for example
	IgnoreDuration time.Duration `json:"ignore_duration,omitempty"`

	// Insecure TLS Skip Verify.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`

	// Client is the http.Client with which to make
	// requests. If not set, DefaultHTTPClient is
	// used.
	Client *http.Client `json:"-"`

	// Headers contains headers to added to the request
	// that is sent for the check
	Headers http.Header `json:"headers,omitempty"`

	// Headers contains headers to added to the request
	// that is sent for the check
	BasicAuth map[string]string `json:"basic_auth,omitempty"`

	// Set degraded instead of down
	Degraded bool `json:"degraded,omitempty"`
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c HTTPChecker) Check() (Result, error) {
	if c.Attempts < 1 {
		c.Attempts = 1
	}
	if c.Retries < 1 {
		c.Retries = 0
	}
	if c.Client == nil {
		c.Client = DefaultHTTPClient(c.InsecureSkipVerify)
	}
	if c.UpStatus == 0 {
		c.UpStatus = http.StatusOK
	}

	result := Result{Title: c.Name, Endpoint: c.URL, Timestamp: Timestamp()}

	result.Times = c.doChecks()

	return c.conclude(result), nil
}

// doChecks executes and returns each attempt.
func (c HTTPChecker) doChecks() Attempts {
	checks := make(Attempts, c.Attempts)
	for i := 0; i < c.Attempts; i++ {
		start := time.Now()
		// check
		err := c.doCheck()
		if err != nil {
			// retries
			if c.Retries > 0 {
				err = c.doRetries()
				if err != nil {
					checks[i].Error = err.Error()
				} else {
					checks[i].RTT = time.Since(start)
				}
			} else {
				checks[i].Error = err.Error()
			}
		} else {
			checks[i].RTT = time.Since(start)
		}
		if c.AttemptSpacing > 0 {
			time.Sleep(c.AttemptSpacing)
		}
	}
	return checks
}

// doRetries executes retries and returns last error.
func (c HTTPChecker) doRetries() error {
	j := 1
	for {
		if c.RetrySpacing > 0 {
			time.Sleep(c.RetrySpacing)
		}
		err := c.doCheck()
		if j >= c.Retries || err == nil {
			return err
		}
		j++
	}
}

// doCheck executes check and returns error.
func (c HTTPChecker) doCheck() error {
	// recreate http request to run dns resolution for each iteration
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return err
	}
	if c.Headers != nil {
		for key, header := range c.Headers {
			evalEnv, _ := envsubst.EvalEnv(strings.Join(header, ", "))
			req.Header.Add(key, evalEnv)
		}
	}
	basicAuthUsername, basicAuthUsernameOk := c.BasicAuth["username"]
	basicAuthPassword, basicAuthPasswordOk := c.BasicAuth["password"]
	if basicAuthUsernameOk && basicAuthPasswordOk {
		username, _ := envsubst.EvalEnv(basicAuthUsername)
		password, _ := envsubst.EvalEnv(basicAuthPassword)
		req.SetBasicAuth(username, password)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = c.checkDown(resp)
	if err != nil {
		return err
	}
	return nil
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
// It detects degraded (high-latency) responses and makes
// the conclusion about the result's status.
func (c HTTPChecker) conclude(result Result) Result {
	result.ThresholdRTT = c.ThresholdRTT

	if len(c.IgnoreTimes) > 0 && c.IgnoreDuration > 0 {
		timeZone := os.Getenv("TZ")
		if timeZone == "" {
			timeZone = "Europe/Paris"
		}
		location, _ := time.LoadLocation(timeZone)
		now := time.Now().UTC().In(location)
		for i := range c.IgnoreTimes {
			start, _ := time.ParseInLocation("15:04:05", c.IgnoreTimes[i], location)
			start = start.AddDate(now.Year(), int(now.Month())-1, now.Day()-1)
			start = start.In(location)
			end := start.Add(c.IgnoreDuration)
			if now.After(start) && now.Before(end) {
				result.Healthy = true
				return result
			}
		}
	}

	// Check errors (down)
	for i := range result.Times {
		if result.Times[i].Error != "" {
			if c.Degraded {
				result.Degraded = true
			} else {
				result.Down = true
			}
			return result
		}
	}

	// Check round trip time (degraded)
	if c.ThresholdRTT > 0 {
		stats := result.ComputeStats()
		if stats.Median > c.ThresholdRTT {
			result.Notice = fmt.Sprintf("median round trip time exceeded threshold (%s)", c.ThresholdRTT)
			result.Degraded = true
			return result
		}
	}

	result.Healthy = true
	return result
}

// checkDown checks whether the endpoint is down based on resp and
// the configuration of c. It returns a non-nil error if down.
// Note that it does not check for degraded response.
func (c HTTPChecker) checkDown(resp *http.Response) error {
	// Check status code
	if resp.StatusCode != c.UpStatus {
		return fmt.Errorf("response status %s", resp.Status)
	}

	// Check response body
	if c.MustContain == "" && c.MustNotContain == "" {
		return nil
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	body := string(bodyBytes)
	if c.MustContain != "" && !strings.Contains(body, c.MustContain) {
		return fmt.Errorf("response does not contain '%s'", c.MustContain)
	}
	if c.MustNotContain != "" && strings.Contains(body, c.MustNotContain) {
		return fmt.Errorf("response contains '%s'", c.MustNotContain)
	}

	return nil
}

func DefaultHTTPClient(insecureSkipVerify bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 0,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   1,
			DisableCompression:    true,
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
}
