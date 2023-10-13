package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

var (
	buildCommit string
	buildDate   string
)

var (
	conf     = flag.StringP("conf", "c", "./config.json", "Config file path")
	pLogPath = flag.StringP("log", "l", "", "Log file path")
	onetime  = flag.BoolP("onetime", "o", false, "Run just once")
	usedns   = flag.BoolP("usedns", "u", false, "Use dns to check your domain record")
	nolog    = flag.BoolP("nolog", "n", false, "Do not log to any file")
	help     = flag.BoolP("help", "h", false, "Print help")
	version  = flag.BoolP("version", "v", false, "Print version")
)

var config Config

var Sources = map[string][]Source{
	"A": {
		&CFTrace{EndPoint: "cf-ns.com", IPFamily: 4},
		&CFTrace{EndPoint: "162.159.36.1", IPFamily: 4},
		&SimpleSource{EndPoint: "https://v4.ident.me/", IPFamily: 4},
		&CFTrace{EndPoint: "1.1.1.1", IPFamily: 4},
		&SimpleSource{EndPoint: "https://api4.ipify.org/", IPFamily: 4},
	},

	"AAAA": {
		&CFTrace{EndPoint: "cf-ns.com", IPFamily: 6},
		&CFTrace{EndPoint: "[2606:4700:4700::1111]", IPFamily: 6},
		&SimpleSource{EndPoint: "https://v6.ident.me/", IPFamily: 6},
		&CFTrace{EndPoint: "[2606:4700:4700::64]", IPFamily: 6},
		&SimpleSource{EndPoint: "https://api6.ipify.org/", IPFamily: 6},
	},
}

var getIPFunc func(string, string) (string, error) = GetCFRecordIP

func init() {
	log.SetOutput(os.Stderr)
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("CloudFlare DDns go %s, built at %s\n", buildCommit, buildDate)
		os.Exit(0)
	}

	if *usedns {
		getIPFunc = GetIPByDns
	}

	if !*nolog && (!*onetime || *pLogPath != "") {
		var logPath string

		if *pLogPath != "" {
			logPath = *pLogPath
		} else if PathExists("/tmp") {
			logPath = "/tmp/cfddns.log"
		} else {
			cwd, _ := os.Getwd()
			logPath = fmt.Sprintf("%s/cfddns.log", cwd)
		}

		logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			log.Printf("[ERR] Failed to open log file %s : %s", logPath, err.Error())
			os.Exit(1)
		}
		log.Printf("[INF] Log to file: %s", logPath)
		mw := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(mw)
	}

	var err error
	if c, err := os.ReadFile(*conf); err == nil {
		if err := json.Unmarshal(c, &config); err == nil {
			if !*onetime {
				log.Println("\n\n[INF] ================ Config loaded ================")
			}
			goto NEXT
		}
	}
	log.Printf("[ERR] Failed to load config: %s", err.Error())
	os.Exit(1)

NEXT:
	ParseConfig(&config)

	if len(config.RecordA) == 0 && len(config.RecordAAAA) == 0 && len(config.RecordSubNET6) == 0 {
		log.Printf("[ERR] No domain configured")
		os.Exit(1)
	}
}

func main() {
	if *onetime {
		checkDomains()
		return
	}

	for {
		checkDomains()
		log.Printf("[INF] Sleep 10 minutes")
		time.Sleep(10 * time.Minute)
	}
}

func ParseConfig(config *Config) {
	if config.SubNet6.Prefix < 1 || config.SubNet6.Prefix > 128 {
		log.Printf("[ERR] Invalid prefix length")
		os.Exit(1)
	} else if config.SubNet6.Prefix > 64 {
		log.Printf("[ERR] Prefix length > 64 is not supported")
		os.Exit(1)
	}

	for _, targe := range config.SubNet6.Targets {
		suffixUint128, err := getSuffixUint128(config.SubNet6.Prefix, targe.Suffix)
		if err != nil {
			log.Printf("[ERR] parse suffix failed: %s", err.Error())
		}
		config.SuffixSubNET6 = append(config.SuffixSubNET6, *suffixUint128)
	}

	generateUrlAndHeader(config)
	success := getAllDnsRecordId(config)
	if !success {
		log.Printf("[ERR] Get all dns record id failed")
	}

	for _, record := range config.RecordA {
		log.Printf("[INF] Watch domain: %s, type A", record.Name)
	}
	for _, record := range config.RecordAAAA {
		log.Printf("[INF] Watch domain: %s, type AAAA", record.Name)
	}
	for index, record := range config.RecordSubNET6 {
		log.Printf("[INF] Watch domain: %s, type AAAA for suffix %s", record.Name, config.SubNet6.Targets[index].Suffix)
	}
}

func getAllDnsRecordId(config *Config) bool {
	for _, domain := range config.A {
		recordId, success := tryGetDnsRecordId(domain, "A")
		if !success {
			return success
		}
		config.RecordA = append(config.RecordA, CFRecord{domain, recordId})
	}

	for _, domain := range config.AAAA {
		recordId, success := tryGetDnsRecordId(domain, "AAAA")
		if !success {
			return success
		}
		config.RecordAAAA = append(config.RecordAAAA, CFRecord{domain, recordId})
	}

	for _, target := range config.SubNet6.Targets {
		recordId, success := tryGetDnsRecordId(target.Domain, "AAAA")
		if !success {
			return success
		}
		config.RecordSubNET6 = append(config.RecordSubNET6, CFRecord{target.Domain, recordId})
	}

	return true
}

func checkDomains() {
	log.Printf("[INF] Checking your domains")
	var currentIP string
	var success bool

	currentIP, success = tryGetCurrentIP("A")
	if !success {
		return
	}
	log.Printf("[INF] Current ipv4 address is: %s", currentIP)
	for _, record := range config.RecordA {
		recordIP, success := tryGetDomainRecordedIP(record.Name, "A")
		if success {
			log.Printf("[INF] Domain %s recorded ipv4 is: %s", record.Name, recordIP)
			if currentIP != recordIP {
				log.Printf("[INF] IPv4 address changed")
				tryUpdateCFRecord(record, "A", currentIP)
			}
		}
	}

	if len(config.RecordAAAA) > 0 || len(config.RecordSubNET6) > 0 {
		currentIP, success = tryGetCurrentIP("AAAA")
		if !success {
			return
		}
		log.Printf("[INF] Current ipv6 address is: %s", currentIP)
	}

	for _, record := range config.RecordAAAA {
		recordIP, success := tryGetDomainRecordedIP(record.Name, "AAAA")
		if success {
			log.Printf("[INF] Domain %s recorded ipv6 is: %s", record.Name, recordIP)
			if currentIP != recordIP {
				log.Printf("[INF] IPv6 address changed")
				tryUpdateCFRecord(record, "AAAA", currentIP)
			}
		}
	}

	for index, record := range config.RecordSubNET6 {
		generatedIPv6, err := genIPv6AddressBySuffix(currentIP, config.SubNet6.Prefix, config.SuffixSubNET6[index])
		if err != nil {
			log.Printf("[WRN] Failed to generate ipv6 address: %s", err.Error())
			continue
		}
		log.Printf("[INF] Generate ipv6 address: %s", generatedIPv6)

		recordIP, success := tryGetDomainRecordedIP(record.Name, "AAAA")
		if success {
			log.Printf("[INF] Domain %s recorded ipv6 is: %s", record.Name, recordIP)
			if generatedIPv6 != recordIP {
				log.Printf("[INF] IPv6 address changed")
				tryUpdateCFRecord(record, "AAAA", generatedIPv6)
			}
		}
	}
}

func tryGetDomainRecordedIP(domain string, recordType string) (string, bool) {
	for i := 0; i < 5; i++ {
		ip, err := getIPFunc(domain, recordType)
		if err == nil {
			return ip, true
		}
		log.Printf("[WRN] %d/5 Failed to resolve domain %s, type %s : %s", i+1, domain, recordType, err.Error())
	}

	log.Printf("[WRN] Resolve domain retry limitation reached")
	return "", false
}

func tryGetCurrentIP(recordType string) (string, bool) {
	for i := 0; i < 5; i++ {
		for _, source := range Sources[recordType] {
			ip, err := source.Fetch()
			if err == nil {
				return ip, true
			}
			log.Printf("[WRN] Failed to get current IP by using %s : %s", i+1, source.String(), err.Error())
		}
		log.Printf("[WRN] %d/5 Failed to get current IP after using all sources", i+1)
	}

	log.Printf("[WRN] Get current IP retry limitation reached")
	return "", false
}

func tryUpdateCFRecord(domain CFRecord, recordType string, ip string) {
	for i := 0; i < 5; i++ {
		err := UpdateCFRecord(domain, recordType, ip)
		if err == nil {
			log.Printf("[INF] Domain %s, type %s record updated to %s", domain.Name, recordType, ip)
			return
		}
		log.Printf("[WRN] %d/5 Failed to update domain %s, type %s: %s", i+1, domain.Name, recordType, err.Error())
	}

	log.Printf("[WRN] Update domain retry limitation reached")
}

func tryGetDnsRecordId(domain string, recordType string) (string, bool) {
	for i := 0; i < 5; i++ {
		recordId, err := getDnsRecordId(domain, recordType)
		if err == nil {
			return recordId, true
		}
		log.Printf("[WRN] %d/5 Failed to get dns record id of domain %s type %s : %s", i+1, domain, recordType, err.Error())
	}

	return "", false
}
