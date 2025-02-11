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

// 最小限
type PostQueryParams struct {
	SQL   string   `json:"sql"`
	PARAM []string `json:"params"`
}
type CreateD1DatabaseParams struct {
	Name string `json:"name"`
}
type Success struct {
}
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type Message struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type CreateDatabaseResponse struct {
	Result   D1Database `json:"result"`
	Errors   []Error    `json:"errors"`
	Messages []Message  `json:"messages"`
	Success  bool       `json:"success"`
}
type QueryResponse struct {
	ERRORS   []Error   `json:"errors"`
	MESSAGES []Message `json:"messages"`
	SUCCESS  bool      `json:"success"`
}

func PostQuery(client *http.Client, query string, CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, CFDBNAME string, CFDBID string) error {
  if (CFACCOUNTID == "") {
    log.Fatal("CFACCOUNTID is Empty")
    return errors.New("CFACCOUNTID is Empty P")
  }
  if (CFEMAIL == "") {
    log.Fatal("CFEMAIL is Empty P")
  }
  if (CFAPIKEY == "") {
    log.Fatal("CFAPIKEY is Empty P")
  }
  if (CFDBNAME == "") {
    log.Fatal("CFDBNAME is Empty P")
  }
  if (CFDBID == "") {
    log.Fatal("CFDBID is Empty P")
  }

	reqBody := PostQueryParams{SQL: query}

	reqBodyJson, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal("Failed to create requiest param")
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database/%s/query", CFACCOUNTID, CFDBID), bytes.NewBuffer(reqBodyJson))
	if err != nil {
		log.Fatal("Failed to Create Request Param for Create Table", err)
		return err
	}

	header := http.Header{}
	header.Add("Content-Type", "application/json")
	header.Add("X-Auth-Email", CFEMAIL)
	header.Add("X-Auth-Key", CFAPIKEY)
	req.Header = header

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("Failed to Post for Create Table")
		return err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var result QueryResponse
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return err
	}
	if !result.SUCCESS {
		log.Fatal("Failed to Post Query, query:", query)
    //log.Fatalf("AccountID:%s, CFEMAIL:%s, CFAPIKEY:%s, CFDBNAME:%s, CFDBID:%s", CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME, CFDBID)
	}

	return nil
}

func D1Init(CFAPIKEY string, CFEMAIL string, CFACCOUNTID string, CFDBNAME string) (string, error) {
	client := &http.Client{}
	client.Timeout = time.Second * 15
	CFDBID, err := findDBID(client, CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME)
	if err != nil {
		log.Fatal("Failed to Find DBID")
		return "", err
	}

	// データベースがなければ作る
	if CFDBID == "" {
		log.Println("Cloudflare D1 has no Database, so create now")
		err := createD1Database(client, CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME)
		if err != nil {
			log.Fatal("Failed to Create Database", err)
			return "", err
		}

		CFDBID, err = findDBID(client, CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME)
		if err != nil {
			log.Fatal("Failed to Find DBID", err)
			return "", err
		}
		if CFDBID == "" {
			log.Fatal("Failed to Find DBID2", err)
			return "", errors.New("Failed to Find DBID2")
		}

		createTableQuery := "CREATE TABLE MESSAGE (ID INTEGER PRIMARY KEY AUTOINCREMENT, MessageID TEXT, AuthorID TEXT, MessageCreatedAT TEXT, CreatedAT TEXT, UpdatedAT TEXT)"
		err = PostQuery(client, createTableQuery, CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME, CFDBID)
		if err != nil {
			log.Fatal("Failed to Create Table", err)
			return "", err
		}
	}

	return CFDBID, nil
}

func RecordMessage(CFDBID string, CFAPIKEY string, CFEMAIL string, CFACCOUNTID string, CFDBNAME string, MessageID string, AuthorID string, MessageCreatedAT time.Time) error {
    client := &http.Client{}
    client.Timeout = time.Second * 15
    layout := "2006-01-02 15:04:05"
    currentTime := time.Now().Format(layout)
    // MessageCreatedATも文字列に変換
    msgCreatedAtStr := MessageCreatedAT.Format(layout)
    query := fmt.Sprintf("INSERT INTO MESSAGE (MessageID, AuthorID, MessageCreatedAT, CreatedAT, UpdatedAT) VALUES ('%s', '%s', '%s', '%s', '%s')", MessageID, AuthorID, msgCreatedAtStr, currentTime, currentTime)
    err := PostQuery(client, query, CFACCOUNTID, CFEMAIL, CFAPIKEY, CFDBNAME, CFDBID)
    if err != nil {
        log.Fatal("Failed to Record Message Query")
        return err
    }
    return nil
}

func findDBID(client *http.Client, CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, CFDBNAME string) (string, error) {
	header := http.Header{}
	header.Add("X-Auth-Email", CFEMAIL)
	header.Add("X-Auth-Key", CFAPIKEY)

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database", CFACCOUNTID), nil)
	if err != nil {
		return "", err
	}
	req.Header = header

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var result ListD1Response
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return "", err
	}

	CFDBID := ""
	for _, v := range result.Result {
		if v.Name == CFDBNAME {
			CFDBID = v.UUID
			break
		}
	}
	return CFDBID, nil
}

func createD1Database(client *http.Client, CFACCOUNTID string, CFEMAIL string, CFAPIKEY string, CFDBNAME string) error {
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
	if err != nil {
		log.Fatal("Failed to create requiest param")
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
