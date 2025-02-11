package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type GetR2BucketResponse struct {
	Result struct {
		CreationData string `json:"creation_date"`
		Location     string `json:"location"`
		Name         string `json:"name"`
		StorageClass string `json:"storage_class"`
	} `json:"result"`
	Errors   []Error  `json:"errors"`
	Messages []string `json:"messages"`
	Success  bool     `json:"success"`
}

func R2Init(CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, BuketName string) error {
	client := &http.Client{}
	exist, err := isR2BucketExist(CFACCOUNTID, CFEMAIL, CFAPIKEY, BuketName)
	if err != nil {
		log.Println("Failed to check R2 bucket existence", err)
		return err
	}
	if !exist {
    log.Print("no exist bucket")
		err := createR2Bucket(client, CFACCOUNTID, CFEMAIL, CFAPIKEY, BuketName)
		if err != nil {
			log.Println("Failed to create R2 bucket", err)
		}
	}
	return nil
}

func createR2Bucket(client *http.Client, CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, BuketName string) error {
  log.Println("createR2Bucket now")
	header := http.Header{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets", CFACCOUNTID), nil)
	if err != nil {
		log.Fatal("Failed to create request")
		return err
	}
	header.Add("X-Auth-Email", CFEMAIL)
	header.Add("X-Auth-Key", CFAPIKEY)
	header.Add("Content-Type", "application/json")
	req.Header = header
	reqBody := map[string]string{
		"name": BuketName,
	}
	reqBodyJson, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("Failed to create request body")
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(reqBodyJson))
	res, err := client.Do(req)
	if err != nil {
		log.Println("Failed to send request")
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("Failed to read response body")
		return err
	}
	var result GetR2BucketResponse
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		log.Println("Failed to parse response body")
		return err
	}
	if !result.Success {
		log.Fatal("Failed to create R2 bucket")
		for _, e := range result.Errors {
			log.Printf("Error: %d, %s", e.Code, e.Message)
		}
		return errors.New("Failed to create R2 bucket")
	}
	return nil
}

func isR2BucketExist(CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, BuketName string) (bool, error) {
	client := &http.Client{}
	header := http.Header{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets/%s", CFACCOUNTID, BuketName), nil)
	if err != nil {
		log.Fatal("Failed to create request")
		return false, err
	}
	header.Add("X-Auth-Email", CFEMAIL)
	header.Add("X-Auth-Key", CFAPIKEY)
	req.Header = header
	res, err := client.Do(req)
	if err != nil {
		log.Fatal("Failed to send request")
		return false, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Failed to read response body")
		return false, err
	}
	var result GetR2BucketResponse
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		log.Fatal("Failed to parse response body")
		return false, err
	}
	if !result.Success {
		log.Printf("Failed to get R2 bucket")
		// errorsのなかにcode:10006がある場合は、バケットが存在しないことを示すためのエラー
		for _, e := range result.Errors {
			if e.Code == 10006 {
				return false, nil
			}
		}
		return false, errors.New("Failed to get R2 bucket")
	}
	return true, nil
}
