# http-probe
A simple and quick URL probing tool (Beta)

> ⚠️ **Beta Status**: This tool is currently in beta phase and might not work as expected.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Input Methods](#input-methods)
  - [Example Usage](#example-usage)
- [Default Behavior](#default-behavior)

## Features
- By default, probe for **status code**, **content-length**, **title**, **redirect chain**, **CSP** header, response time, **Server** and **Powered-By** header (to detect CDN & other technologies).
- By default, Test supported HTTP methods for each URL.
- Fast to use and efficient.
- Anti-Feature: not quite customizable, instead designed for quick usage.
- by default, fallback from https to http.
- Supports domains and URLs as input.

## Installation
```bash
go install github.com/GraveSIN/http-probe@latest
```

## Usage

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

### Example Usage

1. Probe multiple domains:
```bash
http-probe -u "google.com,facebook.com,twitter.com"
```

2. Probe from a file with mixed URLs and FQDNs:
```bash
echo -e "https://google.com\nfacebook.com\nhttp://twitter.com" > urls.txt
http-probe -f urls.txt
```

3. Pipeline with other tools:
```bash
subfinder -d example.com | http-probe
```

## Default Behavior
- Automatically attempts HTTPS first, falls back to HTTP if unsuccessful
- Probes redirect locations
- Probes supported HTTP methods
- Probes html title
- Shows server technology information when available

> Note: This tool is designed for quick recon, and is not as customizable as similar tools
