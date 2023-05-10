# GoFireProx

This is a port of FireProx to Golang

## Installation

```shell
go install -v github.com/mr-pmillz/gofireprox/cmd/gofireprox@latest
```

## gofireprox as library

Integrate gofireprox with other go programs

```go
package main

import (
	"github.com/mr-pmillz/gofireprox"
)

func main() {
	// <snippet> generate random region psuedo code </snippet>
	region := generateRandomRegion()
	fpClient, err := gofireprox.NewFireProx(&gofireprox.FireProxOptions{
		AccessKey:       "CHANGEME",
		SecretAccessKey: "CHANGEME",
		SessionToken:    "",
		Region:          region,
		Command:         "create",
		APIID:           "",
		URL:             "https://ifconfig.me",
	})
	if err != nil {
		panic(err)
	}
	APIID, proxyURL, err := fpClient.CreateAPI()
	if err != nil {
		panic(err)
	}
	// use the proxyURL for your requests as you would normally with an http.Client. See FireProx Docs for headers etc. X-My-X-Forwarded-For: etc...
	// DoWork...
	// Delete API
	fpClient.DeleteAPI(APIID)
}

```

## Credits ##

- Mike Felch ([ustayready](https://twitter.com/ustayready)) - [FireProx](https://github.com/ustayready/fireprox)