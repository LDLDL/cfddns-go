package main

type Config struct {
	Zones  string   `json:"zones"`
	Email  string   `json:"email"`
	APIKey string   `json:"apikey"`
	A      []string `json:"A"`
	AAAA   []string `json:"AAAA"`

	RecordA    []CFRecord `json:"-"`
	RecordAAAA []CFRecord `json:"-"`
}

type CFRecord struct {
	Name     string
	RecordID string
}

type ListDnsRecordResp struct {
	Success bool `json:"success"`
	Result  []struct {
		Name    string `json:"name"`
		Id      string `json:"id"`
		Content string `json:"content"`
	} `json:"result"`
}

type UpdateDnsRecordReq struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type UpdateDnsRecordResp struct {
	Success bool `json:"success"`
}
