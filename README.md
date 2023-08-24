# CloudFlare DDns
Cloudflare DDNS written in go

# How to use
## 0. Add domain record on Cloudflare web panel
The domain should be added manually before running this program  
While adding domain, input a incorrect ip.(later you can check if this program updating your domain well or not)  

## 1. Config file example
```
{
    "zones": "{zone id}",
    "email": "{email}",
    "apikey": "{global apikey}",

    "A": [
        "example.com",
        "4.example.com"
    ],
    "AAAA": [
        "example.com",
        "6.example.com"
    ]
}
```

## 2. Executable arguments
- --conf    Config file path, default value is ./conf.json  
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
