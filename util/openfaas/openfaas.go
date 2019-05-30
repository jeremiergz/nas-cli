package openfaas

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

const (
	// ParseMediaTitle calls media parsing function
	ParseMediaTitle = "parse-media-title"
)

// InvokeFaaS invokes FaaS gateway for given function with given body and returns response as a byte array
func InvokeFaaS(function string, body []byte) ([]byte, error) {
	openFaaSConfig := viper.GetStringMapString("openfaas")
	netClient := &http.Client{
		Timeout: time.Second * 5,
	}
	url := fmt.Sprintf("%s/%s", openFaaSConfig["url"], function)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.SetBasicAuth(openFaaSConfig["username"], openFaaSConfig["password"])

	resp, err := netClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	rawData, _ := ioutil.ReadAll(resp.Body)

	return rawData, nil
}
