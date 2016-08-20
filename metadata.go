package metadata

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"

	"github.com/dutchcoders/metadata/json"
	"github.com/op/go-logging"
)

type InstanceIdentity struct {
	AccountID          string `json:"accountId"`
	Architecture       string `json:"architecture"`
	AvailabilityZone   string `json:"availabilityZone"`
	BillingProducts    string `json:"billingProducts"`
	DevpayProductCodes string `json:"devpayProductCodes"`
	ImageID            string `json:"imageId"`
	InstanceID         string `json:"instanceId"`
	InstanceType       string `json:"instanceType"`
	KernelID           string `json:"kernelId"`
	PendingTime        string `json:"pendingTime"`
	PrivateIp          string `json:"privateIp"`
	RamdiskID          string `json:"ramdiskId"`
	Region             string `json:"region"`
	Version            string `json:"version"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
var log = logging.MustGetLogger("metadata")

var DefaultClient = New()

type MetaData struct {
	*Client
}

type Dynamic struct {
	*Client
}

type Client struct {
	Client  *http.Client
	BaseURL *url.URL

	MetaData *MetaData
	Dynamic  *Dynamic
}

func (d *Dynamic) InstanceIdentity() (*InstanceIdentity, error) {
	var ii InstanceIdentity
	if req, err := d.NewRequest("GET", "/latest/dynamic/instance-identity/document", nil); err != nil {
		return nil, err
	} else if err := d.Do(req, &ii); err != nil {
		return nil, err
	} else {
		return &ii, nil
	}
}

func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	var buf io.Reader
	if body == nil {
	} else if v, ok := body.(io.Reader); ok {
		buf = v
	} else if v, ok := body.(json.M); ok {
		buf = new(bytes.Buffer)
		if err := json.NewEncoder(buf.(io.ReadWriter)).Encode(v); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("not supported type: %s", reflect.TypeOf(body))
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/json; charset=UTF-8")
	req.Header.Add("Accept", "text/json")
	return req, nil
}

func New() *Client {
	baseURL, err := url.Parse("http://169.254.169.254/")
	if err != nil {
		panic(err)
	}

	c := &Client{
		Client:  http.DefaultClient,
		BaseURL: baseURL,
	}

	c.MetaData = &MetaData{c}
	c.Dynamic = &Dynamic{c}

	return c
}

func (wd *Client) do(req *http.Request, v interface{}) error {
	if b, err := httputil.DumpRequest(req, true); err == nil {
		log.Debug(string(b))
	}

	resp, err := wd.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if b, err := httputil.DumpResponse(resp, true); err == nil {
		log.Debug(string(b))
	}

	var r io.Reader = resp.Body

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < 300 {
	} else if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Not found")
	} else {
		return fmt.Errorf("Unknown error.")
	}

	switch v := v.(type) {
	case io.Writer:
		io.Copy(v, resp.Body)
	case interface{}:
		return json.NewDecoder(r).Decode(&v)
	}

	return nil
}

func (wd *Client) Do(req *http.Request, v interface{}) error {
	return wd.do(req, v)
}

/*
metadata
    .Dynamic()
    .InstanceIdentity()

metadata
    .Dynamic
    .InstanceIdentity()


curl http://169.254.169.254/latest/dynamic/instance-identity/document^C
*/
