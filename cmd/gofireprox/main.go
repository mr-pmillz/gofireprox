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
	"time"
)

func main() {
	accessKey := flag.String("access_key", "", "AWS Access Key")
	secretAccessKey := flag.String("secret_access_key", "", "AWS Secret Access Key")
	sessionToken := flag.String("session_token", "", "AWS Session Token.")
	profile := flag.String("profile", "", "AWS Profile Section to use.")
	region := flag.String("region", "", "AWS Region")
	command := flag.String("command", "", "Commands: list, create, delete, update")
	apiID := flag.String("api_id", "", "API ID")
	proxyURL := flag.String("url", "", "URL end-point")
	verbose := flag.Bool("verbose", false, "toggles verbosity to reduce API requests that fetch additional verbose data")
	version := flag.Bool("version", false, "show version of gofireprox")
	apiCacheDuration := flag.Duration("cache-duration", 60*time.Second, "sets api list cache duration in seconds. Ex. -cache-duration 120s for 120 seconds or 2m for 2 minutes")
	flag.Parse()

	fpOptions := &gofireprox.FireProxOptions{
		AccessKey:       *accessKey,
		SecretAccessKey: *secretAccessKey,
		SessionToken:    *sessionToken,
		Profile:         *profile,
		Region:          *region,
		Command:         *command,
		APIID:           *apiID,
		URL:             *proxyURL,
		Verbose:         *verbose,
		Version:         *version,
		CacheDuration:   *apiCacheDuration,
	}

	if fpOptions.Version {
		fmt.Printf("GoFireprox Current Version: %s\n", gofireprox.CurrentVersion)
		os.Exit(0)
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
