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

var conf = flag.StringP("conf", "c", "./config.json", "Config file path")
var pLogPath = flag.StringP("log", "l", "", "Log file path")
var onetime = flag.BoolP("onetime", "o", false, "Run just once")
var usedns = flag.BoolP("usedns", "u", false, "Use dns to check your domain record")
var nolog = flag.BoolP("nolog", "n", false, "Do not log to any file")
var help = flag.BoolP("help", "h", false, "Print help")

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
			log.Printf("[ERR] Failed opening log file %s : %s", logPath, err.Error())
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
				log.Println("[INF] Config loaded")
			}
			goto NEXT
		}
	}
	log.Printf("[ERR] Failed loading config: %s", err.Error())
	os.Exit(1)

NEXT:
	ParseConfig(&config)

	if len(config.RecordA) == 0 && len(config.RecordAAAA) == 0 {
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
		time.Sleep(10 * time.Minute)
		log.Printf("[INF] Sleep 10 minutes")
	}
}

func checkDomains() {
	log.Printf("[INF] Checking your domains")

	if len(config.RecordA) > 0 {
		currentIP, success := tryGetCurrentIP("A")
		if !success {
			return
		}
		log.Printf("[INF] Current ipv4 address is: %s", currentIP)
		for _, record := range config.RecordA {
			recordIP, success := tryGetDomainRecordedIP(record.Name, "A")
			if success {
				log.Printf("[INF] domain '%s' recorded ipv4 is: %s", record.Name, recordIP)
				if currentIP != recordIP {
					tryUpdateCFRecord(record, "A", currentIP)
				}
			}
		}
	}

	if len(config.RecordAAAA) > 0 {
		currentIP, success := tryGetCurrentIP("AAAA")
		if !success {
			return
		}
		log.Printf("[INF] Current ipv4 address is: %s", currentIP)
		for _, record := range config.RecordA {
			recordIP, success := tryGetDomainRecordedIP(record.Name, "AAAA")
			if success {
				log.Printf("[INF] domain '%s' recorded ipv6 is: %s", record.Name, recordIP)
				if currentIP != recordIP {
					tryUpdateCFRecord(record, "AAAA", currentIP)
				}
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
		log.Printf("[WAR] %d/5 Failed to resolve domain '%s' : %s", i+1, domain, err.Error())
	}

	log.Printf("[WAR] Resolve domain retry limitation reached")
	return "", false
}

func tryGetCurrentIP(recordType string) (string, bool) {
	for i := 0; i < 5; i++ {
		for _, source := range Sources[recordType] {
			ip, err := source.Fetch()
			if err == nil {
				return ip, true
			}
			log.Printf("[WAR] %d/5 Failed to get current IP by using %s : %s", i+1, source.String(), err.Error())
		}
	}

	log.Printf("[WAR] Get current IP retry limitation reached")
	return "", false
}

func tryUpdateCFRecord(domain CFRecord, recordType string, ip string) {
	for i := 0; i < 5; i++ {
		err := UpdateCFRecord(domain, recordType, ip)
		if err == nil {
			return
		}
		log.Printf("[WAR] %d/5 Failed to update domain: %s", i+1, err.Error())
	}

	log.Printf("[WAR] Update domain retry limitation reached")
}
