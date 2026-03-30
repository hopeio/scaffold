package esv8

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	jsonx "github.com/hopeio/gox/encoding/json"
	esx "github.com/hopeio/gox/database/elasticsearch"
)

func GetResponseData[T any](response *esapi.Response, err error) (*T, error) {
	defer response.Body.Close()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}
	var res T
	err = jsonx.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func GetSearchResponseData[T any](response *esapi.Response, err error) (*esx.SearchResponse[T], error) {
	return GetResponseData[esx.SearchResponse[T]](response, err)
}

func CreateDocument[T any](ctx context.Context, es *elasticsearch.Client, index, id string, obj T) error {
	body, _ := jsonx.Marshal(obj)
	esreq := esapi.CreateRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
	}
	resp, err := esreq.Do(ctx, es)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
