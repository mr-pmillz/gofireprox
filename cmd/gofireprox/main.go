package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mr-pmillz/gofireprox"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	accessKey := flag.String("access_key", "", "AWS Access Key")
	secretAccessKey := flag.String("secret_access_key", "", "AWS Secret Access Key")
	sessionToken := flag.String("session_token", "", "AWS Session Token")
	region := flag.String("region", "", "AWS Region")
	command := flag.String("command", "", "Commands: list, create, delete, update")
	apiID := flag.String("api_id", "", "API ID")
	proxyURL := flag.String("url", "", "URL end-point")
	flag.Parse()

	fpOptions := &gofireprox.FireProxOptions{
		AccessKey:       *accessKey,
		SecretAccessKey: *secretAccessKey,
		SessionToken:    *sessionToken,
		Region:          *region,
		Command:         *command,
		APIID:           *apiID,
		URL:             *proxyURL,
	}

	fp, err := gofireprox.NewFireProx(fpOptions)
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		fmt.Printf("CTRL+C detected: %+v\nCleaning up...", s)
		fp.Cleanup()
		os.Exit(1)
	}()

	switch *command {
	case "list":
		_, err = fp.ListAPIs()
		if err != nil {
			log.Println("[ERROR] Failed to list APIs:", err)
			os.Exit(1)
		}
	case "delete":
		successful := fp.DeleteAPI(aws.ToString(apiID))
		var success string
		if successful {
			success = "Success!"
		} else {
			success = "Failed!"
		}
		fmt.Printf("Deleting %s => %s\n", fp.Options.APIID, success)
	case "create":
		if _, _, err = fp.CreateAPI(); err != nil {
			log.Fatal(err)
		}
	case "update":
		successful, err := fp.UpdateAPI(fp.Options.APIID, fp.Options.URL)
		if err != nil {
			log.Fatal(err)
		}
		var success string
		if successful {
			success = "Success!"
		} else {
			success = "Failed!"
		}
		fmt.Printf("API Update Complete: %s\n", success)
	default:
		fmt.Printf("[ERROR] Unsupported command: %s\n", *command)
		os.Exit(1)
	}
}
