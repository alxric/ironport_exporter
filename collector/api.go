package collector

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/common/log"
)

//APICall is what we use to query the Ironport API
func APICall(target *target) ([]byte, error) {
	var defaultTransport http.RoundTripper = &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   4 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          30,
		IdleConnTimeout:       3 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{Transport: defaultTransport}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/xml/status", target.Hostname), nil)
	req.SetBasicAuth(target.Username, target.Password)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer resp.Body.Close()
	return body, nil
}
