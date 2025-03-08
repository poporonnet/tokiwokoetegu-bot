package cloudflare

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

// CloudflareAPI は Cloudflare API とのやり取りを行う構造体
type CloudflareAPI struct {
	Client    *http.Client
	AccountID string
	Email     string
	APIKey    string
	DBName    string
	DBID      string
}

// D1Database は Cloudflare D1 データベースの情報を表す構造体
type D1Database struct {
	Name      string     `json:"name"`
	NumTables int        `json:"num_tables"`
	UUID      string     `json:"uuid"`
	Version   string     `json:"version"`
	CreatedAt *time.Time `json:"created_at"`
	FileSize  int64      `json:"file_size"`
}

// ListD1Response は D1 データベースのリスト取得 API の応答を表す構造体
type ListD1Response struct {
	Result []D1Database `json:"result"`
}

// PostQueryParams は SQL クエリ実行 API のパラメータを表す構造体
type PostQueryParams struct {
	SQL    string   `json:"sql"`
	Params []string `json:"params,omitempty"`
}

// CreateD1DatabaseParams は D1 データベース作成 API のパラメータを表す構造体
type CreateD1DatabaseParams struct {
	Name string `json:"name"`
}

// APIMessage は API の応答に含まれるメッセージを表す構造体
type APIMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CreateDatabaseResponse は D1 データベース作成 API の応答を表す構造体
type CreateDatabaseResponse struct {
	Result   D1Database   `json:"result"`
	Errors   []APIMessage `json:"errors"`
	Messages []APIMessage `json:"messages"`
	Success  bool         `json:"success"`
}

// QueryResponse は SQL クエリ実行 API の応答を表す構造体
type QueryResponse struct {
	Errors   []APIMessage `json:"errors"`
	Messages []APIMessage `json:"messages"`
	Success  bool         `json:"success"`
}

// NewCloudflareAPI は CloudflareAPI の新しいインスタンスを作成する
func NewCloudflareAPI(accountID, email, apiKey, dbName string) (*CloudflareAPI, error) {
	if accountID == "" {
		return nil, errors.New("Cloudflare アカウント ID が指定されていません")
	}
	if email == "" {
		return nil, errors.New("Cloudflare メールアドレスが指定されていません")
	}
	if apiKey == "" {
		return nil, errors.New("Cloudflare API キーが指定されていません")
	}
	if dbName == "" {
		return nil, errors.New("データベース名が指定されていません")
	}

	client := &http.Client{
		Timeout: time.Second * 15,
	}

	return &CloudflareAPI{
		Client:    client,
		AccountID: accountID,
		Email:     email,
		APIKey:    apiKey,
		DBName:    dbName,
	}, nil
}

// PostQuery は SQL クエリを実行する
func (api *CloudflareAPI) PostQuery(query string) error {
	reqBody := PostQueryParams{SQL: query}
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("リクエストパラメータの JSON 化に失敗しました: %w", err)
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database/%s/query", api.AccountID, api.DBID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		return fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Email", api.Email)
	req.Header.Add("X-Auth-Key", api.APIKey)

	res, err := api.Client.Do(req)
	if err != nil {
		return fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("レスポンスの読み込みに失敗しました: %w", err)
	}

	var result QueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("クエリの実行に失敗しました: %s", query)
	}

	return nil
}

func D1Init(apiKey, email, accountID, dbName string) (string, error) {
	api, err := NewCloudflareAPI(accountID, email, apiKey, dbName)
	if err != nil {
		return "", fmt.Errorf("Cloudflare API クライアントの初期化に失敗しました: %w", err)
	}

	// データベース ID を検索
	dbID, err := api.findDBID()
	if err != nil {
		return "", fmt.Errorf("データベース ID の検索に失敗しました: %w", err)
	}

	// データベースがなければ作成
	if dbID == "" {
		log.Println("Cloudflare D1 データベースが存在しないため、新規作成します")
		err := api.createD1Database()
		if err != nil {
			return "", fmt.Errorf("データベースの作成に失敗しました: %w", err)
		}

		// 再度データベース ID を検索
		dbID, err = api.findDBID()
		if err != nil {
			return "", fmt.Errorf("作成したデータベースの ID 検索に失敗しました: %w", err)
		}

		if dbID == "" {
			return "", errors.New("データベースが作成されましたが、ID の取得に失敗しました")
		}

		// API インスタンスに ID を設定
		api.DBID = dbID

		// テーブルの作成
		createTableQuery := `
			CREATE TABLE MESSAGE (
				ID INTEGER PRIMARY KEY AUTOINCREMENT,
				MessageID TEXT,
				MessageContent TEXT,
				Attachments TEXT,
				AuthorID TEXT,
				MessageCreatedAT TEXT,
				CreatedAT TEXT,
				UpdatedAT TEXT
			)
		`
		err = api.PostQuery(createTableQuery)
		if err != nil {
			return "", fmt.Errorf("テーブルの作成に失敗しました: %w", err)
		}
	} else {
		// データベース ID を設定
		api.DBID = dbID
	}

	return dbID, nil
}


func (api *CloudflareAPI) findDBID() (string, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database", api.AccountID),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Add("X-Auth-Email", api.Email)
	req.Header.Add("X-Auth-Key", api.APIKey)

	res, err := api.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンスの読み込みに失敗しました: %w", err)
	}

	var result ListD1Response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}

	// 指定された名前のデータベースを検索
	for _, db := range result.Result {
		if db.Name == api.DBName {
			return db.UUID, nil
		}
	}

	// 見つからない場合は空文字を返す
	return "", nil
}

func (api *CloudflareAPI) createD1Database() error {
	reqBody := CreateD1DatabaseParams{Name: api.DBName}
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("リクエストパラメータの JSON 化に失敗しました: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database", api.AccountID),
		bytes.NewBuffer(reqBodyJSON),
	)
	if err != nil {
		return fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Email", api.Email)
	req.Header.Add("X-Auth-Key", api.APIKey)

	res, err := api.Client.Do(req)
	if err != nil {
		return fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("レスポンスの読み込みに失敗しました: %w", err)
	}

	var result CreateDatabaseResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}

	if !result.Success {
		return errors.New("データベースの作成に失敗しました")
	}

	log.Println("データベースの作成が完了しました")
	return nil
}
