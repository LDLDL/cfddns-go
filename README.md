# CloudFlare DDns
Cloudflare DDNS written in go

# How to use
## 0. Add domain record on Cloudflare web panel
The domain should be added manually before running this program  
While adding domain, input a incorrect ip.(later you can check if this program updating your domain well or not)  

## 1. Config file example
```
{
    "zones": "",
    "email": "",
    "apikey": "",

    "A": [
        "example.com",
        "4.example.com"
    ],
    "AAAA": [
        "example.com",
        "6.example.com"
    ],

    "subnet6": {
        "prefix": 64,
        "targets": [
            {
                "domain": "6.example.com",
                "suffix": "::1111:2222:3333:ffff/64"
            },
            {
                "domain": "66.example.com",
                "suffix": "::8888/16"
            }
        ]
    }
}
```
zones: your cloudflare domain zone id  
email: your cloudflare account email  
apikey: your cloudflare global api key  

A: domain list to update A record to this machine  
AAAA: domain list to update AAAA record to this machine  

subnet6: (this is optional)  
prefix: your ipv6 prefix length  
targets: your other machines in your subnet to update  
domain: domain to update AAAA record to your other machine  
suffix: your other machine's ipv6 suffix  

## 2. Executable arguments
- --conf    Config file path, default value is ./config.json  
- --log     Log file path  
- --onetime Run just once, suitable for Crontab.  
- --usedns  Use dns to get domain current recorded ip instead of Cloudflare api  
- --nolog   Do not log to any file, only log to console  

## 3. Linux systemd service
Copy executable and config file to ```/opt/cfddns-go/```  

Create service file at ```/usr/lib/systemd/system``` with following content  
```
[Unit]
Description=Cloudflare DDns go
After=network.target

[Service]
Type=simple
User=root
ExecStart=/opt/cfddns-go/cfddns-go --conf /opt/cfddns-go/config.json
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```
Start and enable service
```
systemctl start cfddns-go
systemctl enable cfddns-go
```
