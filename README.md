# http-probe
A simple and quick URL probing tool (Beta)

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Input Methods](#input-methods)
  - [Example Usage](#example-usage)
- [Default Behavior](#default-behavior)

## Features
- By default, probe for **status code**, **content-length**, **title**, **redirect chain**, **CSP** header, response time, **Server** and **Powered-By** header (to detect CDN & other technologies).
- Fast to use and efficient.
- Anti-Feature: not quite customizable, instead designed for quick usage.
- by default, fallback from https to http.
- Supports domains and URLs as input.

## Installation
```bash
go install github.com/GraveSIN/http-probe@latest
```

## Usage

```
Probe a URL via different HTTP methods

Usage:
  http-probe [flags]

Flags:
  -d, --data string     HTTP request body data
  -f, --file string     File containing URLs (one per line)
  -h, --help            help for http-probe
  -X, --method string   HTTP method to use (default: GET) (default "GET")
  -o, --output string   Output file path
  -t, --threads int     Number of concurrent threads (default 10)
  -T, --timeout int     Timeout in seconds for each request (default: 3) (default 3)
  -u, --url strings     Target URL(s) to probe
```

### Input Methods
1. Via Command Line:
```bash
http-probe -u google.com,facebook.com,http://facebook.com
```

2. Via File:
```bash
http-probe -f urls.txt
```

3. Via Standard Input (stdin):
```bash
cat urls.txt | http-probe
```
or
```bash
echo "google.com" | http-probe
```

## Default Behavior
- Automatically attempts HTTPS first, falls back to HTTP if unsuccessful
- Probes redirect locations
- Probes html title
- Shows server technology information when available
