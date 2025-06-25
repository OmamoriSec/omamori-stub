# Omamori: Protection from unholy

## Setup

```bash
apt install dnsutils -y
git clone https://github.com/Cosmicoppai/omamori.git
cd omamori/app  
go build -o omamori .
./omamori
```

Server runs on `127.0.0.1:2053` by default.

## Test with

```bash
dig +noedns @127.0.0.1 -p 2053 google.com
```

## Config

- `conf` - Upstream DNS server server (default: Cloudflare + OpenDNS`)
- `blocked_file.txt` - auto-downloads StevenBlack hosts list

## Features

- Domain Blocking via radix tree 
- DNS resolution with configurable upstream servers
- LRU cache (1000 entries, 5s cleanup interval)
- Supports all standard DNS record types
