package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type FireProx struct {
	Options *FireProxOptions
	Client  *apigateway.Client
}

type FireProxOptions struct {
	AccessKey       string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Command         string
	APIID           string
	URL             string
}

// NewFireProx ...
func NewFireProx(opts *FireProxOptions) (*FireProx, error) {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(opts.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretAccessKey, opts.SessionToken)),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := apigateway.NewFromConfig(cfg)

	fp := &FireProx{
		Options: &FireProxOptions{
			AccessKey:       opts.AccessKey,
			SecretAccessKey: opts.SecretAccessKey,
			SessionToken:    opts.SessionToken,
			Region:          cfg.Region,
			Command:         opts.Command,
			APIID:           opts.APIID,
			URL:             opts.URL,
		},
		Client: client,
	}
	return fp, nil
}

func (fp *FireProx) cleanup() {
	fmt.Println("\n\n\n\n[+] Cleaning up")
	items, err := fp.listAPIs()
	if err != nil {
		log.Println("Error listing APIs, make sure your aws config/account is properly configured with the appropriate permissions.")
	}
	for _, item := range items {
		input := &apigateway.DeleteRestApiInput{
			RestApiId: item.Id,
		}
		_, err = fp.Client.DeleteRestApi(context.TODO(), input)
		if err != nil {
			log.Println("[ERROR] Failed to delete API:", item.Id)
		}
	}
	fmt.Println()
}

func main() {
	accessKey := flag.String("access_key", "", "AWS Access Key")
	secretAccessKey := flag.String("secret_access_key", "", "AWS Secret Access Key")
	sessionToken := flag.String("session_token", "", "AWS Session Token")
	region := flag.String("region", "", "AWS Region")
	command := flag.String("command", "", "Commands: list, create, delete, update")
	apiID := flag.String("api_id", "", "API ID")
	proxyURL := flag.String("url", "", "URL end-point")
	flag.Parse()

	fpOptions := &FireProxOptions{
		AccessKey:       *accessKey,
		SecretAccessKey: *secretAccessKey,
		SessionToken:    *sessionToken,
		Region:          *region,
		Command:         *command,
		APIID:           *apiID,
		URL:             *proxyURL,
	}

	fp, err := NewFireProx(fpOptions)
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		fmt.Printf("CTRL+C detected: %+v\nCleaning up...", s)
		fp.cleanup()
		os.Exit(1)
	}()

	switch *command {
	case "list":
		_, err = fp.listAPIs()
		if err != nil {
			log.Println("[ERROR] Failed to list APIs:", err)
			os.Exit(1)
		}
	case "delete":
		successful := fp.deleteAPI(aws.ToString(apiID))
		var success string
		if successful {
			success = "Success!"
		} else {
			success = "Failed!"
		}
		fmt.Printf("Deleting %s => %s\n", fp.Options.APIID, success)
	case "create":
		if _, err = fp.createAPI(); err != nil {
			log.Fatal(err)
		}
	case "update":
		successful, err := fp.updateAPI(fp.Options.APIID, fp.Options.URL)
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

// getResources ...
func (fp *FireProx) getResources(apiID string) (string, error) {
	resourceInput := apigateway.GetResourcesInput{
		RestApiId: aws.String(apiID),
	}
	resp, err := fp.Client.GetResources(context.TODO(), &resourceInput)
	if err != nil {
		return "", err
	}
	for _, item := range resp.Items {
		if aws.ToString(item.Path) == "/{proxy+}" {
			return aws.ToString(item.Id), nil
		}
	}
	return "", nil
}

// getIntegration ...
func (fp *FireProx) getIntegration(apiID string) (string, error) {
	resourceID, err := fp.getResources(apiID)
	if err != nil {
		return "", err
	}
	integrationInput := apigateway.GetIntegrationInput{
		HttpMethod: aws.String("ANY"),
		ResourceId: &resourceID,
		RestApiId:  aws.String(apiID),
	}
	resp, err := fp.Client.GetIntegration(context.TODO(), &integrationInput)
	if err != nil {
		return "", err
	}

	return aws.ToString(resp.Uri), nil
}

// listAPIs ...
func (fp *FireProx) listAPIs() ([]types.RestApi, error) {
	input := &apigateway.GetRestApisInput{}

	resp, err := fp.Client.GetRestApis(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	apiIDs := make([]string, len(resp.Items))
	for i, item := range resp.Items {
		proxyURL, err := fp.getIntegration(aws.ToString(item.Id))
		if err != nil {
			return nil, err
		}
		proxyURL = strings.ReplaceAll(proxyURL, "{proxy}", "")
		proxiedURL := fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/fireprox/", aws.ToString(item.Id), fp.Options.Region)
		fmt.Printf("[%s] (%s) %v: %s => %s\n", item.CreatedDate, aws.ToString(item.Id), item.Name, proxiedURL, proxyURL)
		apiIDs[i] = *item.Id
	}

	return resp.Items, nil
}

// deleteAPI ...
func (fp *FireProx) deleteAPI(apiItem string) bool {
	items, err := fp.listAPIs()
	if err != nil {
		log.Println("Error listing APIs, make sure your aws config/account is properly configured with the appropriate permissions.")
		return false
	}
	for _, item := range items {
		if apiItem == *item.Id {
			input := &apigateway.DeleteRestApiInput{
				RestApiId: item.Id,
			}
			_, err = fp.Client.DeleteRestApi(context.TODO(), input)
			if err != nil {
				log.Println("[ERROR] Failed to delete API:", item.Id)
			}
			return true
		}
	}
	return false
}

type templateInfo struct {
	Version string
	Title   string
}

// newTemplateInfo ...
func (fp *FireProx) newTemplateInfo() (*templateInfo, error) {
	title, err := url.Parse(fp.Options.URL)
	if err != nil {
		return nil, err
	}
	fireProxTitle := fmt.Sprintf("fireprox_%s", title.Hostname())
	versionDate := time.Now().Format("2006-01-02T15:04:05Z")
	return &templateInfo{
		Version: fireProxTitle,
		Title:   versionDate,
	}, nil
}

// getTemplate ...
func (fp *FireProx) getTemplate(tmplInfo *templateInfo) (*apigateway.ImportRestApiInput, error) {
	// Snippet from: https://github.com/ustayready/fireprox/blob/master/fire.py
	tmpl := `{
		"swagger": "2.0",
		"info": {
		  "version": "` + tmplInfo.Version + `",
		  "title": "` + tmplInfo.Title + `"
		},
		"basePath": "/",
		"schemes": [
		  "https"
		],
		"paths": {
		  "/": {
			"get": {
			  "parameters": [
				{
				  "name": "proxy",
				  "in": "path",
				  "required.": true,
				  "type": "string"
				},
				{
				  "name": "X-My-X-Forwarded-For",
				  "in": "header",
				  "required": false,
				  "type": "string"
				}
			  ],
			  "responses": {},
			  "x-amazon-apigateway-integration": {
				"uri": "` + fp.Options.URL + `/",
				"responses": {
				  "default": {
					"statusCode": "200"
				  }
				},
				"requestParameters": {
				  "integration.request.path.proxy": "method.request.path.proxy",
				  "integration.request.header.X-Forwarded-For": "method.request.header.X-My-X-Forwarded-For"
				},
				"passthroughBehavior": "when_no_match",
				"httpMethod": "ANY",
				"cacheNamespace": "irx7tm",
				"cacheKeyParameters": [
				  "method.request.path.proxy"
				],
				"type": "http_proxy"
			  }
			}
		  },
		  "/{proxy+}": {
			"x-amazon-apigateway-any-method": {
			  "produces": [
				"application/json"
			  ],
			  "parameters": [
				{
				  "name": "proxy",
				  "in": "path",
				  "required": true,
				  "type": "string"
				},
				{
				  "name": "X-My-X-Forwarded-For",
				  "in": "header",
				  "required": false,
				  "type": "string"
				}
			  ],
			  "responses": {},
			  "x-amazon-apigateway-integration": {
				"uri": "` + fp.Options.URL + `/{proxy}",
				"responses": {
				  "default": {
					"statusCode": "200"
				  }
				},
				"requestParameters": {
				  "integration.request.path.proxy": "method.request.path.proxy",
				  "integration.request.header.X-Forwarded-For": "method.request.header.X-My-X-Forwarded-For"
				},
				"passthroughBehavior": "when_no_match",
				"httpMethod": "ANY",
				"cacheNamespace": "irx7tm",
				"cacheKeyParameters": [
				  "method.request.path.proxy"
				],
				"type": "http_proxy"
			  }
			}
		  }
		}
	  }`

	params := make(map[string]string)
	params["endpointConfigurationTypes"] = "REGIONAL"
	ir := &apigateway.ImportRestApiInput{
		Parameters: params,
		Body:       []byte(tmpl),
	}
	return ir, nil
}

// createAPI ...
func (fp *FireProx) createAPI() (string, error) {
	tmplInfo, err := fp.newTemplateInfo()
	if err != nil {
		return "", err
	}
	irAPI, err := fp.getTemplate(tmplInfo)
	if err != nil {
		return "", err
	}
	resp, err := fp.Client.ImportRestApi(context.TODO(), irAPI)
	if err != nil {
		return "", err
	}

	createDeploymentInput := &apigateway.CreateDeploymentInput{
		RestApiId:        resp.Id,
		StageDescription: aws.String("GoFireProx Prod"),
		StageName:        aws.String("GoFireProx"),
		Description:      aws.String("GoFireProx Production Deployment"),
	}

	_, err = fp.Client.CreateDeployment(context.TODO(), createDeploymentInput)
	if err != nil {
		return "", err
	}

	return aws.ToString(resp.Id), nil
}

// updateAPI ...
func (fp *FireProx) updateAPI(apiID, apiURL string) (bool, error) {
	resourceID, err := fp.getResources(apiID)
	if err != nil {
		return false, err
	}
	if resourceID != "" {
		fmt.Printf("Found resource %s for %s\n", resourceID, apiID)
	}
	updateIntegrationInput := &apigateway.UpdateIntegrationInput{
		HttpMethod: aws.String("ANY"),
		ResourceId: aws.String(resourceID),
		RestApiId:  aws.String(apiID),
		PatchOperations: []types.PatchOperation{{
			From:  nil,
			Op:    "replace",
			Path:  aws.String("/uri"),
			Value: aws.String(fmt.Sprintf("%s/{proxy}", apiURL)),
		}},
	}

	resp, err := fp.Client.UpdateIntegration(context.TODO(), updateIntegrationInput)
	if err != nil {
		return false, err
	}

	log.Printf("API updated with ID: %v\n", apiID)
	return strings.ReplaceAll(aws.ToString(resp.Uri), "{proxy}", "") == apiURL, nil
}
