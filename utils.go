package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
}

var clientIPv4 = &http.Client{
	Timeout: 30 * time.Second,
}
var dialer4 net.Dialer

var clientIPv6 = &http.Client{
	Timeout: 30 * time.Second,
}
var dialer6 net.Dialer

func init() {
	transport4 := http.DefaultTransport.(*http.Transport).Clone()
	transport4.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer4.DialContext(ctx, "tcp4", addr)
	}
	clientIPv4.Transport = transport4

	transport6 := http.DefaultTransport.(*http.Transport).Clone()
	transport6.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer6.DialContext(ctx, "tcp6", addr)
	}
	clientIPv6.Transport = transport6
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func GetIPByDns(domain string, recordType string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var network string
	if recordType == "A" {
		network = "ip4"
	} else if recordType == "AAAA" {
		network = "ip6"
	} else {
		return "", fmt.Errorf("record type %s is not supported", recordType)
	}

	addrs, err := net.DefaultResolver.LookupIP(ctx, network, domain)
	if err != nil {
		return "", err
	}

	return addrs[0].String(), nil
}

func httpGetRequest(url string, header http.Header, family int) (resp *http.Response, err error) {
	var httpClient *http.Client = client
	if family == 4 {
		httpClient = clientIPv4
	} else if family == 6 {
		httpClient = clientIPv6
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if header != nil {
		req.Header = header
	}
	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET '%s' Status: %s", url, resp.Status)
	}

	return resp, nil
}

func cfapiPutRequest(url string, body []byte) (resp *http.Response, err error) {
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header = cfHeader
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("PUT '%s' Status: %s", url, resp.Status)
	}

	return resp, nil
}
