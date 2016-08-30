package metadata

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
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
	PrivateIP          string `json:"privateIp"`
	RamdiskID          string `json:"ramdiskId"`
	Region             string `json:"region"`
	Version            string `json:"version"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
var log = logging.MustGetLogger("metadata")

var DefaultClient = New()

type metaData struct {
	*Client
}

type dynamic struct {
	*Client
}

type Client struct {
	Client *http.Client

	BaseURL *url.URL
}

func MetaData() *metaData {
	return DefaultClient.MetaData()
}

func Dynamic() *dynamic {
	return DefaultClient.Dynamic()
}

func (m *metaData) PublicHostName() (string, error) {
	var s string
	if req, err := m.NewRequest("GET", "/latest/meta-data/public-hostname", nil); err != nil {
		return "", err
	} else if err := m.Do(req, &s); err != nil {
		return "", err
	} else {
		return s, nil
	}
}

func (m *metaData) PublicIP() (net.IP, error) {
	var s string
	if req, err := m.NewRequest("GET", "/latest/meta-data/public-ipv4", nil); err != nil {
		return nil, err
	} else if err := m.Do(req, &s); err != nil {
		return nil, err
	} else {
		ip := net.ParseIP(s)
		return ip, nil
	}
}

func (d *dynamic) InstanceIdentity() (*InstanceIdentity, error) {
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

func (c *Client) MetaData() *metaData {
	return &metaData{c}
}

func (c *Client) Dynamic() *dynamic {
	return &dynamic{c}
}

func New() *Client {
	baseURL, err := url.Parse("http://169.254.169.254/")
	if err != nil {
		panic(err)
	}

	return &Client{
		Client:  http.DefaultClient,
		BaseURL: baseURL,
	}
}

func (c *Client) do(req *http.Request, v interface{}) error {
	if b, err := httputil.DumpRequest(req, true); err == nil {
		log.Debug(string(b))
	}

	resp, err := c.Client.Do(req)
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
	case *string:
		if b, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		} else {
			*v = string(b)
		}
	case io.Writer:
		io.Copy(v, resp.Body)
	case interface{}:
		return json.NewDecoder(r).Decode(&v)
	}

	return nil
}

func (c *Client) Do(req *http.Request, v interface{}) error {
	return c.do(req, v)
}
