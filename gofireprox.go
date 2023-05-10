package gofireprox

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"log"
	"net/url"
	"strings"
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
	var region string
	if opts.Region == "" {
		region = "us-east-1"
	} else {
		region = opts.Region
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
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

// Cleanup ...
func (fp *FireProx) Cleanup() {
	fmt.Println("\n\n\n\n[+] Cleaning up")
	items, err := fp.ListAPIs()
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
	versionDate := time.Now().Format("2006-01-02 15:04:05")
	return &templateInfo{
		Version: versionDate,
		Title:   fireProxTitle,
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
	return &apigateway.ImportRestApiInput{
		Parameters: params,
		Body:       []byte(tmpl),
	}, nil
}

// createDeployment ...
func (fp *FireProx) createDeployment(apiID *string) (string, string, error) {
	createDeploymentInput := &apigateway.CreateDeploymentInput{
		RestApiId:        apiID,
		StageDescription: aws.String("FireProx Prod"),
		StageName:        aws.String("fireprox"),
		Description:      aws.String("FireProx Production Deployment"),
	}

	resp, err := fp.Client.CreateDeployment(context.TODO(), createDeploymentInput)
	if err != nil {
		return "", "", err
	}

	return aws.ToString(resp.Id), fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/fireprox/", aws.ToString(apiID), fp.Options.Region), nil
}

// storeAPI ...
func (fp *FireProx) storeAPI(apiID, name, createdAT, targetURL, proxyURL string) {
	fmt.Printf("[%v] (%s) %s %s => %s\n", createdAT, apiID, name, proxyURL, targetURL)
}

// ListAPIs ...
func (fp *FireProx) ListAPIs() ([]types.RestApi, error) {
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
		fmt.Printf("[%s] (%s) %s: %s => %s\n", item.CreatedDate.String(), aws.ToString(item.Id), aws.ToString(item.Name), proxiedURL, proxyURL)
		apiIDs[i] = *item.Id
	}

	return resp.Items, nil
}

// DeleteAPI ...
func (fp *FireProx) DeleteAPI(apiID string) bool {
	items, err := fp.ListAPIs()
	if err != nil {
		log.Println("Error listing APIs, make sure your aws config/account is properly configured with the appropriate permissions.")
		return false
	}
	for _, item := range items {
		if apiID == *item.Id {
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

// CreateAPI ...
func (fp *FireProx) CreateAPI() (string, error) {
	fmt.Printf("Creating => %s...\n", fp.Options.URL)
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

	_, proxyURL, err := fp.createDeployment(resp.Id)
	if err != nil {
		return "", err
	}
	// apiID string, name string, createdAT string, targetURL string, proxyURL string
	fp.storeAPI(aws.ToString(resp.Id), tmplInfo.Title, resp.CreatedDate.String(), fp.Options.URL, proxyURL)

	return aws.ToString(resp.Id), nil
}

// UpdateAPI ...
func (fp *FireProx) UpdateAPI(apiID, apiURL string) (bool, error) {
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