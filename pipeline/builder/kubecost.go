package builder

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

const (
	kubecostURL              = "https://kubecost.namely.land"
	kubecostDefaultWindow    = "7d"
	errKubecostAPI           = "failed to request sizing from kubecost with status code: %s"
	kubecostTimeoutInSeconds = 180
)

var (
	// defaultProfilePerAccount translates spinnaker account to kubecost profile
	defaultProfilePerAccount = map[string]string{
		"int":            "development",
		"int-k8s":        "development",
		"staging":        "production",
		"staging-k8s":    "production",
		"production":     "production",
		"production-k8s": "production",
		"ops":            "production",
		"ops-k8s":        "production",
	}

	profiles = map[string]struct {
		p                    float32
		targetCPUUtilization float32
		targetRAMUtilization float32
	}{
		"development": {
			p:                    0.85,
			targetCPUUtilization: 0.8,
			targetRAMUtilization: 0.8,
		},
		"production": {
			p:                    0.98,
			targetCPUUtilization: 0.65,
			targetRAMUtilization: 0.65,
		}, "high-availability": {
			p:                    0.999,
			targetCPUUtilization: 0.5,
			targetRAMUtilization: 0.5,
		},
	}
)

// getKubecostSizing gets sizing recommendations json from kubecost
func getKubecostSizing(profile string, window string) ([]byte, error) {
	url, err := url.Parse(kubecostURL + "/model/savings/requestSizing")
	if err != nil {
		return nil, err
	}
	q := url.Query()
	q.Set("p", fmt.Sprintf("%f", profiles[profile].p))
	q.Set("window", window)
	q.Set("targetCPUUtilization", fmt.Sprintf("%f", profiles[profile].targetCPUUtilization))
	q.Set("targetRAMUtilization", fmt.Sprintf("%f", profiles[profile].targetRAMUtilization))
	url.RawQuery = q.Encode()
	s := url.String()
	return httpGetWithTimeoutAndRetry(s, kubecostTimeoutInSeconds*time.Second, 2)
}
func httpGetWithTimeoutAndRetry(url string, timeout time.Duration, retries int) ([]byte, error) {
	httpClient := http.Client{
		Timeout: timeout,
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		if retries > 0 {
			return httpGetWithTimeoutAndRetry(url, timeout, retries-1)
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if retries > 0 {
			return httpGetWithTimeoutAndRetry(url, timeout, retries-1)
		}
		return nil, fmt.Errorf(errKubecostAPI, resp.Status)
	}
	//We Read the response body on the line below.
	return ioutil.ReadAll(resp.Body)
}

// requests saves the name and recommended CPU and RAM for each requests
type requests struct {
	requestsCPU float64
	requestsRAM float64
}

// getRecommendedCPUAndRAM filters kubecost requestSizing json and returns a map where the key is the controller name
// and the value is a map where the key is the container name and the value is the recommended RAM and CPU
func getRecommendedCPUAndRAM(json []byte) map[string]map[string]requests {
	recommendations := make(map[string]map[string]requests)
	controllers := gjson.GetBytes(json, "controllers")
	if !controllers.Exists() {
		return nil
	}
	//loop over every controller
	controllers.ForEach(func(k gjson.Result, ctrl gjson.Result) bool {
		name := ctrl.Get("name").String()
		containers := ctrl.Get("containers")
		recommendations[name] = make(map[string]requests)
		// loop over every resources
		containers.ForEach(func(key gjson.Result, cont gjson.Result) bool {
			c := requests{
				requestsCPU: cont.Get("requests.cpu").Float(),
				requestsRAM: cont.Get("requests.ram").Float(),
			}
			recommendations[name][key.String()] = c
			return true // keep iterating
		})
		return true // keep iterating
	})
	return recommendations
}

// GetKubecostData gets kubecost recommended sizing for each profile
func GetKubecostData() (map[string][]byte, error) {
	kubecostData := make(map[string][]byte)
	wg := sync.WaitGroup{}
	mutex := &sync.Mutex{}
	err := make(chan error, len(profiles))
	for profile := range profiles {
		wg.Add(1)
		go func(profile string) {
			bytes, reqErr := getKubecostSizing(profile, kubecostDefaultWindow)
			err <- reqErr
			mutex.Lock()
			kubecostData[profile] = bytes
			mutex.Unlock()
			wg.Done()
		}(profile)
	}
	wg.Wait()
	close(err)
	for kubeErr := range err {
		if kubeErr != nil {
			return nil, kubeErr
		}
	}
	return kubecostData, nil
}
