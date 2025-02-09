package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type D1Database struct {
	Name      string     `json:"name"`
	NumTables int        `json:"num_tables"`
	UUID      string     `json:"uuid"`
	Version   string     `json:"version"`
	CreatedAt *time.Time `json:"created_at"`
	FileSize  int64      `json:"file_size"`
}

type ListD1Response struct {
	Result []D1Database `json:"result"`
}

func D1Init(CFAPIKEY string, CFEMAIL string, CFACCOUNTID string, CFDBNAME string) (err error) {
	client := &http.Client{}

	client.Timeout = time.Second * 15

	header := http.Header{}
	header.Add("X-Auth-Email", CFEMAIL)
	header.Add("X-Auth-Key", CFAPIKEY)

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database", CFACCOUNTID), nil)
	if err != nil {
		return err
	}
	req.Header = header

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
  var result ListD1Response
  if err := json.Unmarshal([]byte(body), &result); err != nil {
    return err
  }
  
  var isThere = false
  for _, v := range(result.Result) {
    if v.Name == CFDBNAME {
      isThere = true
      break
    }
  }
  if !isThere {
    log.Println("Cloudflare D1 has no Database, so create now")
    err := createD1Database(client, CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME)
    if err != nil {
      log.Fatal("Failed to Create Database", err)
      return err
    }
  }
  return nil
}

type CreateD1DatabaseParams struct {
	Name string `json:"name"`
}
type Success struct {
}
type Error struct {
  Code int `json:"code"`
  Message string `json:"message"`
}
type Message struct {
  Code int `json:"code"`
  Message string `json:"message"`
}
type CreateDatabaseResponse struct {
	Result D1Database `json:"result"`
  Errors []Error `json:"errors"`
  Messages []Message `json:"messages"`
  Success bool `json:"success"`
}
func createD1Database(client *http.Client, CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, CFDBNAME string) (err error) {
  header := http.Header{}
  header.Add("Content-Type", "application/json")
  header.Add("X-Auth-Email", CFEMAIL)
  header.Add("X-Auth-Key", CFAPIKEY)
  reqBody := CreateD1DatabaseParams{Name: CFDBNAME}
  reqBodyJson, err := json.Marshal(reqBody)
  if err != nil {
    log.Fatal("Failed to create request body json")
    return err
  }
  req, err := http.NewRequest("POST", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database", CFACCOUNTID), bytes.NewBuffer(reqBodyJson))
  req.Header = header
  
  res, err := client.Do(req)
  if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
  var result CreateDatabaseResponse
  if err := json.Unmarshal([]byte(body), &result); err != nil {
    log.Fatal("Failed to Unmarshal result")
    return err
  }
  if !result.Success {
    log.Fatal("Failed to Create Database")
    return errors.New("Failed to Create Database")
  }
  log.Println("Complete to Create Database")
  return nil
}
