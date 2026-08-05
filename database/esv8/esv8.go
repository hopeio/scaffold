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

// GetResponseData reads and decodes an Elasticsearch API response body into T.
// It closes the response body and returns an error for non-200 status codes.
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

// GetSearchResponseData decodes an Elasticsearch search response into a typed SearchResponse wrapper.
func GetSearchResponseData[T any](response *esapi.Response, err error) (*esx.SearchResponse[T], error) {
	return GetResponseData[esx.SearchResponse[T]](response, err)
}

// CreateDocument serializes obj as JSON and creates an Elasticsearch document with the given index and ID.
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
