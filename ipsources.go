package main

import (
	"fmt"
	"io"
	"strings"
)

type Source interface {
	Fetch() (string, error)
	String() string
}

type SimpleSource struct {
	EndPoint string
	IPFamily int
}

func (s *SimpleSource) Fetch() (string, error) {
	resp, err := httpGetRequest(s.EndPoint, nil, s.IPFamily)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func (s *SimpleSource) String() string {
	return fmt.Sprintf("<simple source; endpoint='%s', IPFamily='%d'>", s.EndPoint, s.IPFamily)
}

type CFTrace struct {
	EndPoint string
	IPFamily int
}

func (s *CFTrace) Fetch() (string, error) {
	resp, err := httpGetRequest(fmt.Sprintf("https://%s/cdn-cgi/trace", s.EndPoint), nil, s.IPFamily)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyText := strings.TrimSpace(string(body))
	for _, line := range strings.Split(bodyText, "\n") {
		if strings.HasPrefix(line, "ip=") {
			return strings.TrimSpace(line[3:]), nil
		}
	}

	return "", fmt.Errorf("no ip provided in cloudflare trace info")
}

func (s *CFTrace) String() string {
	return fmt.Sprintf("<CFtrace; endpoint='%s', IPFamily='%d'>", s.EndPoint, s.IPFamily)
}
