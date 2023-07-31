package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var cfHeader http.Header
var cfDnsApiUrl string

func ParseConfig(config *Config) {
	generateUrlAndHeader(config)
	getAllDnsRecordId(config)

	for _, record := range config.RecordA {
		log.Printf("[INF] Watch domain: %s, type A", record.Name)
	}
	for _, record := range config.RecordAAAA {
		log.Printf("[INF] Watch domain: %s, type AAAA", record.Name)
	}
}

func GetCFRecordIP(domain string, recordType string) (string, error) {
	var result ListDnsRecordResp

	resp, err := httpGetRequest(fmt.Sprintf("%s?type=%s&name=%s", cfDnsApiUrl, recordType, domain), cfHeader, 0)
	if err != nil {
		goto ERROR
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	if err != nil {
		goto ERROR
	}

	if !result.Success {
		err = fmt.Errorf("cloudflare api return unsuccess")
		goto ERROR
	}

	for _, record := range result.Result {
		if record.Name == domain {
			return record.Content, nil
		}
	}

	err = fmt.Errorf("domain record content not found")
ERROR:
	return "", err
}

func UpdateCFRecord(domain CFRecord, recordType string, ip string) error {
	var result UpdateDnsRecordResp

	data := UpdateDnsRecordReq{
		Type:    recordType,
		Name:    domain.Name,
		Content: ip,
		TTL:     60,
		Proxied: false,
	}
	jsonString, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := cfapiPutRequest(fmt.Sprintf("%s/%s", cfDnsApiUrl, domain.RecordID), jsonString)
	if err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("cloudflare api return unsuccess")
	}

	return nil
}

func generateUrlAndHeader(config *Config) {
	cfHeader = http.Header{
		"X-Auth-Email":  {config.Email},
		"X-Auth-Key":    {config.APIKey},
		"cache-control": {"no-cache"},
	}

	cfDnsApiUrl = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", config.Zones)
}

func getAllDnsRecordId(config *Config) {
	for _, domain := range config.A {
		config.RecordA = append(config.RecordA, CFRecord{domain, getDnsRecordId(domain, "A")})
	}

	for _, domain := range config.AAAA {
		config.RecordAAAA = append(config.RecordAAAA, CFRecord{domain, getDnsRecordId(domain, "AAAA")})
	}
}

func getDnsRecordId(domain string, recordType string) string {
	var result ListDnsRecordResp

	resp, err := httpGetRequest(fmt.Sprintf("%s?type=%s&name=%s", cfDnsApiUrl, recordType, domain), cfHeader, 0)
	if err != nil {
		goto EXIT
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	if err != nil {
		goto EXIT
	}

	if !result.Success {
		err = fmt.Errorf("cloudflare api return unsuccess")
		goto EXIT
	}

	for _, record := range result.Result {
		if record.Name == domain {
			return record.Id
		}
	}
	err = fmt.Errorf("domain record id not found")

EXIT:
	log.Printf("[ERR] Get domain %s, type %s record id failed: %s", domain, recordType, err.Error())
	os.Exit(1)
	return ""
}
