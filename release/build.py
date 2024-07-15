import os
import glob
import subprocess
import datetime

subprocess.run("rm ./cfddns-go*", shell=True)

commitId = subprocess.run("git rev-parse --short HEAD", shell=True, capture_output=True)
commitId = commitId.stdout.decode().strip()

time = datetime.datetime.now(datetime.UTC).strftime('%Y-%m-%d %H:%M')

t = [['linux', 'amd64'],
     ['linux', 'arm'],
     ['linux', 'arm64'],
     ['linux', 'mips'],
     ['linux', 'mipsle'],
     ['windows', 'amd64'],
]

for o, a in t:
    subprocess.run(
        f'''env GOOS="{o}" GOARCH="{a}" CGO_ENABLED=0 '''
        f'''go build -ldflags="-s -w '''
        f'''-X main.buildCommit={commitId} -X \'main.buildDate={time}\'" '''
        f'''-o cfddns-go_{o}_{a}{".exe" if o == "windows" else ""} '''
        f'''../''', shell=True
    )

bins = glob.glob("cfddns-go*")
for bin in bins:
    if bin.endswith(".exe"):
        subprocess.run(f"zip {bin}.zip {bin}", shell=True)
        os.remove(bin)
    else:
        subprocess.run(f"gzip {bin}", shell=True)

