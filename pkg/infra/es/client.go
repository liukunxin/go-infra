package es

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// Client wraps the official Elasticsearch client with convenience methods.
type Client struct {
	es *elasticsearch.Client
}

// NewClient creates and validates a Client from cfg.
func NewClient(cfg Config) (*Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("es: at least one address is required")
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 100 * time.Millisecond
	}

	// Build a default transport with sensible connection-pool and timeout settings.
	defaultTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	if cfg.EnableTLS {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12} //nolint:gosec
		if len(cfg.CACert) > 0 {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(cfg.CACert) {
				return nil, errors.New("es: failed to parse CA certificate")
			}
			tlsCfg.RootCAs = pool
		}
		defaultTransport.TLSClientConfig = tlsCfg
	}

	esCfg := elasticsearch.Config{
		Addresses:  cfg.Addresses,
		Username:   cfg.Username,
		Password:   cfg.Password,
		APIKey:     cfg.APIKey,
		MaxRetries: cfg.MaxRetries,
		RetryBackoff: func(i int) time.Duration {
			return cfg.RetryBackoff * time.Duration(1<<uint(i-1))
		},
		Transport: defaultTransport,
	}

	raw, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("es: create client: %w", err)
	}

	c := &Client{es: raw}
	if err := c.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("es: ping: %w", err)
	}
	return c, nil
}

// ESClient returns the underlying *elasticsearch.Client for advanced usage.
func (c *Client) ESClient() *elasticsearch.Client { return c.es }

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

// Ping checks cluster reachability.
func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es: ping: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Index management
// ---------------------------------------------------------------------------

// IndexExists reports whether index exists in Elasticsearch.
func (c *Client) IndexExists(ctx context.Context, index string) (bool, error) {
	res, err := c.es.Indices.Exists([]string{index},
		c.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("es: index exists %q: %w", index, err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
}

// CreateIndex creates index with an optional settings/mappings body.
// Pass nil to create with default settings.
func (c *Client) CreateIndex(ctx context.Context, index string, body map[string]interface{}) error {
	opts := []func(*esapi.IndicesCreateRequest){
		c.es.Indices.Create.WithContext(ctx),
	}
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("es: create index %q marshal body: %w", index, err)
		}
		opts = append(opts, c.es.Indices.Create.WithBody(bytes.NewReader(b)))
	}
	res, err := c.es.Indices.Create(index, opts...)
	if err != nil {
		return fmt.Errorf("es: create index %q: %w", index, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// DeleteIndex removes index from Elasticsearch.
func (c *Client) DeleteIndex(ctx context.Context, index string) error {
	res, err := c.es.Indices.Delete([]string{index},
		c.es.Indices.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es: delete index %q: %w", index, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// PutMapping updates the field mapping of an existing index.
func (c *Client) PutMapping(ctx context.Context, index string, mapping map[string]interface{}) error {
	b, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("es: put mapping %q marshal: %w", index, err)
	}
	res, err := c.es.Indices.PutMapping([]string{index}, bytes.NewReader(b),
		c.es.Indices.PutMapping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es: put mapping %q: %w", index, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Document CRUD
// ---------------------------------------------------------------------------

// IndexDoc indexes doc under index/id. If id is empty, ES generates one.
func (c *Client) IndexDoc(ctx context.Context, index, id string, doc interface{}) error {
	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("es: index doc marshal: %w", err)
	}
	opts := []func(*esapi.IndexRequest){
		c.es.Index.WithContext(ctx),
		c.es.Index.WithRefresh("false"),
	}
	if id != "" {
		opts = append(opts, c.es.Index.WithDocumentID(id))
	}
	res, err := c.es.Index(index, bytes.NewReader(b), opts...)
	if err != nil {
		return fmt.Errorf("es: index doc %q/%q: %w", index, id, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// GetDoc fetches the document at index/id and unmarshals _source into dest.
// Returns ErrNotFound when the document does not exist.
func (c *Client) GetDoc(ctx context.Context, index, id string, dest interface{}) error {
	res, err := c.es.Get(index, id, c.es.Get.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es: get doc %q/%q: %w", index, id, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	var envelope struct {
		Source json.RawMessage `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("es: get doc decode: %w", err)
	}
	return json.Unmarshal(envelope.Source, dest)
}

// UpdateDoc performs a partial update (doc-based) on index/id.
func (c *Client) UpdateDoc(ctx context.Context, index, id string, doc interface{}) error {
	payload := map[string]interface{}{"doc": doc}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("es: update doc marshal: %w", err)
	}
	res, err := c.es.Update(index, id, bytes.NewReader(b),
		c.es.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es: update doc %q/%q: %w", index, id, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// DeleteDoc removes the document at index/id.
// Returns ErrNotFound when the document does not exist.
func (c *Client) DeleteDoc(ctx context.Context, index, id string) error {
	res, err := c.es.Delete(index, id, c.es.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es: delete doc %q/%q: %w", index, id, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if res.IsError() {
		return &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// SearchResult holds the parsed results of a Search call.
type SearchResult struct {
	Total int64
	Hits  []SearchHit
}

// SearchHit represents one document in a search response.
type SearchHit struct {
	Index  string
	ID     string
	Score  float64
	Source json.RawMessage
}

// Search executes a DSL query against one or more indices and returns parsed hits.
//
// Example query:
//
//	map[string]interface{}{
//	    "query": map[string]interface{}{
//	        "match": map[string]interface{}{"title": "go"},
//	    },
//	    "size": 10,
//	}
func (c *Client) Search(ctx context.Context, indices []string, query map[string]interface{}) (*SearchResult, error) {
	b, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: search marshal query: %w", err)
	}
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(indices...),
		c.es.Search.WithBody(bytes.NewReader(b)),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: search: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}

	var envelope struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Index  string          `json:"_index"`
				ID     string          `json:"_id"`
				Score  float64         `json:"_score"`
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("es: search decode response: %w", err)
	}

	result := &SearchResult{Total: envelope.Hits.Total.Value}
	for _, h := range envelope.Hits.Hits {
		result.Hits = append(result.Hits, SearchHit{
			Index:  h.Index,
			ID:     h.ID,
			Score:  h.Score,
			Source: h.Source,
		})
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Bulk
// ---------------------------------------------------------------------------

// BulkActionType describes the operation to perform in a bulk request.
type BulkActionType string

const (
	BulkIndex  BulkActionType = "index"
	BulkCreate BulkActionType = "create"
	BulkUpdate BulkActionType = "update"
	BulkDelete BulkActionType = "delete"
)

// BulkAction is a single operation in a Bulk call.
type BulkAction struct {
	Action BulkActionType
	Index  string
	ID     string      // optional; ES auto-generates for index/create if empty
	Doc    interface{} // nil for BulkDelete; for BulkUpdate, the partial doc (not wrapped in "doc")
}

// BulkResult summarises a bulk operation outcome.
type BulkResult struct {
	Took   int               `json:"took"`
	Errors bool              `json:"errors"`
	Items  []json.RawMessage `json:"items"`
}

// Bulk executes multiple index/create/update/delete operations in a single request.
// Returns an error if the HTTP call fails; individual item errors are available in BulkResult.Errors.
func (c *Client) Bulk(ctx context.Context, actions []BulkAction) (*BulkResult, error) {
	if len(actions) == 0 {
		return &BulkResult{}, nil
	}

	var buf bytes.Buffer
	for _, a := range actions {
		meta := map[string]interface{}{
			string(a.Action): buildBulkMeta(a),
		}
		if err := json.NewEncoder(&buf).Encode(meta); err != nil {
			return nil, fmt.Errorf("es: bulk encode meta: %w", err)
		}
		if a.Action != BulkDelete {
			var body interface{}
			if a.Action == BulkUpdate {
				body = map[string]interface{}{"doc": a.Doc}
			} else {
				body = a.Doc
			}
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				return nil, fmt.Errorf("es: bulk encode body: %w", err)
			}
		}
	}

	res, err := c.es.Bulk(bytes.NewReader(buf.Bytes()),
		c.es.Bulk.WithContext(ctx),
		c.es.Bulk.WithRefresh("false"),
	)
	if err != nil {
		return nil, fmt.Errorf("es: bulk request: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, &ResponseError{StatusCode: res.StatusCode, RawBody: readBody(res.Body)}
	}

	var result BulkResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: bulk decode response: %w", err)
	}
	return &result, nil
}

func buildBulkMeta(a BulkAction) map[string]interface{} {
	m := map[string]interface{}{"_index": a.Index}
	if a.ID != "" {
		m["_id"] = a.ID
	}
	return m
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func readBody(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 4096))
	return string(b)
}
