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
	"fmt"
	"github.com/mr-pmillz/gofireprox"
)

func main() {
	region := "us-east-1"
	
	fpClient, err := gofireprox.NewFireProx(&gofireprox.FireProxOptions{
		AccessKey:       "CHANGEME",
		SecretAccessKey: "CHANGEME",
		Region:          region,
		URL:             "https://ifconfig.me",
	})
	if err != nil {
		panic(err)
	}
	
	apiID, proxyURL, err := fpClient.CreateAPI()
	if err != nil {
		panic(err)
	}
	// use the proxyURL for your requests as you would normally with an http.Client. See FireProx Docs for headers etc. X-My-X-Forwarded-For: etc...
	// DoWork...
	// Delete API
	successful := fpClient.DeleteAPI(apiID)
	var success string
	if successful {
		success = "Success!"
	} else {
		success = "Failed!"
	}
	fmt.Printf("Deleting %s => %s\n", apiID, success)
}

```

## Credits ##

- Mike Felch ([ustayready](https://twitter.com/ustayready)) - [FireProx](https://github.com/ustayready/fireprox)