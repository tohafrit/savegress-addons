package fhir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/savegress/healthsync/pkg/models"
)

// Client represents a FHIR R4 client
type Client struct {
	baseURL      string
	httpClient   *http.Client
	authProvider AuthProvider
	version      string
}

// AuthProvider interface for FHIR authentication
type AuthProvider interface {
	GetToken(ctx context.Context) (string, error)
	TokenType() string
}

// ClientConfig holds client configuration
type ClientConfig struct {
	BaseURL      string
	AuthProvider AuthProvider
	Timeout      time.Duration
	Version      string // R4, STU3, etc.
}

// NewClient creates a new FHIR client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	version := config.Version
	if version == "" {
		version = "R4"
	}

	return &Client{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		authProvider: config.AuthProvider,
		version:      version,
	}
}

// doRequest performs an HTTP request to the FHIR server
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")
	req.Header.Set("Accept", "application/fhir+json")

	// Add authorization if provider is set
	if c.authProvider != nil {
		token, err := c.authProvider.GetToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get auth token: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.authProvider.TokenType(), token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var opOutcome OperationOutcome
		if err := json.Unmarshal(respBody, &opOutcome); err == nil {
			return nil, &FHIRError{
				StatusCode: resp.StatusCode,
				Outcome:    &opOutcome,
			}
		}
		return nil, &FHIRError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return respBody, nil
}

// OperationOutcome represents a FHIR OperationOutcome
type OperationOutcome struct {
	ResourceType string                   `json:"resourceType"`
	Issue        []OperationOutcomeIssue `json:"issue"`
}

// OperationOutcomeIssue represents an issue in an OperationOutcome
type OperationOutcomeIssue struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Diagnostics string `json:"diagnostics,omitempty"`
	Details     *models.CodeableConcept `json:"details,omitempty"`
}

// FHIRError represents a FHIR API error
type FHIRError struct {
	StatusCode int
	Message    string
	Outcome    *OperationOutcome
}

func (e *FHIRError) Error() string {
	if e.Outcome != nil && len(e.Outcome.Issue) > 0 {
		return fmt.Sprintf("FHIR error %d: %s - %s", e.StatusCode, e.Outcome.Issue[0].Code, e.Outcome.Issue[0].Diagnostics)
	}
	return fmt.Sprintf("FHIR error %d: %s", e.StatusCode, e.Message)
}

// Read retrieves a resource by type and ID
func (c *Client) Read(ctx context.Context, resourceType, id string) ([]byte, error) {
	path := fmt.Sprintf("/%s/%s", resourceType, id)
	return c.doRequest(ctx, "GET", path, nil)
}

// Create creates a new resource
func (c *Client) Create(ctx context.Context, resourceType string, resource interface{}) ([]byte, error) {
	path := fmt.Sprintf("/%s", resourceType)
	return c.doRequest(ctx, "POST", path, resource)
}

// Update updates an existing resource
func (c *Client) Update(ctx context.Context, resourceType, id string, resource interface{}) ([]byte, error) {
	path := fmt.Sprintf("/%s/%s", resourceType, id)
	return c.doRequest(ctx, "PUT", path, resource)
}

// Delete deletes a resource
func (c *Client) Delete(ctx context.Context, resourceType, id string) error {
	path := fmt.Sprintf("/%s/%s", resourceType, id)
	_, err := c.doRequest(ctx, "DELETE", path, nil)
	return err
}

// Search searches for resources
func (c *Client) Search(ctx context.Context, resourceType string, params SearchParams) (*Bundle, error) {
	queryParams := url.Values{}
	for key, values := range params {
		for _, v := range values {
			queryParams.Add(key, v)
		}
	}

	path := fmt.Sprintf("/%s?%s", resourceType, queryParams.Encode())
	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var bundle Bundle
	if err := json.Unmarshal(respBody, &bundle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bundle: %w", err)
	}

	return &bundle, nil
}

// SearchParams represents FHIR search parameters
type SearchParams map[string][]string

// Bundle represents a FHIR Bundle
type Bundle struct {
	ResourceType string        `json:"resourceType"`
	ID           string        `json:"id,omitempty"`
	Type         string        `json:"type"`
	Total        int           `json:"total,omitempty"`
	Link         []BundleLink  `json:"link,omitempty"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// BundleLink represents a link in a Bundle
type BundleLink struct {
	Relation string `json:"relation"`
	URL      string `json:"url"`
}

// BundleEntry represents an entry in a Bundle
type BundleEntry struct {
	FullURL  string                 `json:"fullUrl,omitempty"`
	Resource json.RawMessage        `json:"resource,omitempty"`
	Search   *BundleEntrySearch     `json:"search,omitempty"`
	Request  *BundleEntryRequest    `json:"request,omitempty"`
	Response *BundleEntryResponse   `json:"response,omitempty"`
}

// BundleEntrySearch represents search information
type BundleEntrySearch struct {
	Mode  string  `json:"mode,omitempty"`
	Score float64 `json:"score,omitempty"`
}

// BundleEntryRequest represents a request in a transaction
type BundleEntryRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

// BundleEntryResponse represents a response in a transaction
type BundleEntryResponse struct {
	Status   string `json:"status"`
	Location string `json:"location,omitempty"`
	Etag     string `json:"etag,omitempty"`
}

// Transaction executes a batch/transaction
func (c *Client) Transaction(ctx context.Context, bundle *Bundle) (*Bundle, error) {
	respBody, err := c.doRequest(ctx, "POST", "/", bundle)
	if err != nil {
		return nil, err
	}

	var result Bundle
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction result: %w", err)
	}

	return &result, nil
}

// GetPatient retrieves a Patient resource
func (c *Client) GetPatient(ctx context.Context, id string) (*models.Patient, error) {
	respBody, err := c.Read(ctx, "Patient", id)
	if err != nil {
		return nil, err
	}

	var patient models.Patient
	if err := json.Unmarshal(respBody, &patient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient: %w", err)
	}

	return &patient, nil
}

// SearchPatients searches for patients
func (c *Client) SearchPatients(ctx context.Context, params SearchParams) ([]*models.Patient, error) {
	bundle, err := c.Search(ctx, "Patient", params)
	if err != nil {
		return nil, err
	}

	patients := make([]*models.Patient, 0, len(bundle.Entry))
	for _, entry := range bundle.Entry {
		var patient models.Patient
		if err := json.Unmarshal(entry.Resource, &patient); err != nil {
			continue
		}
		patients = append(patients, &patient)
	}

	return patients, nil
}

// CreatePatient creates a new patient
func (c *Client) CreatePatient(ctx context.Context, patient *models.Patient) (*models.Patient, error) {
	patient.ResourceType = models.ResourceTypePatient

	respBody, err := c.Create(ctx, "Patient", patient)
	if err != nil {
		return nil, err
	}

	var created models.Patient
	if err := json.Unmarshal(respBody, &created); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created patient: %w", err)
	}

	return &created, nil
}

// GetEncounter retrieves an Encounter resource
func (c *Client) GetEncounter(ctx context.Context, id string) (*models.Encounter, error) {
	respBody, err := c.Read(ctx, "Encounter", id)
	if err != nil {
		return nil, err
	}

	var encounter models.Encounter
	if err := json.Unmarshal(respBody, &encounter); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encounter: %w", err)
	}

	return &encounter, nil
}

// SearchEncounters searches for encounters
func (c *Client) SearchEncounters(ctx context.Context, params SearchParams) ([]*models.Encounter, error) {
	bundle, err := c.Search(ctx, "Encounter", params)
	if err != nil {
		return nil, err
	}

	encounters := make([]*models.Encounter, 0, len(bundle.Entry))
	for _, entry := range bundle.Entry {
		var encounter models.Encounter
		if err := json.Unmarshal(entry.Resource, &encounter); err != nil {
			continue
		}
		encounters = append(encounters, &encounter)
	}

	return encounters, nil
}

// GetObservation retrieves an Observation resource
func (c *Client) GetObservation(ctx context.Context, id string) (*models.Observation, error) {
	respBody, err := c.Read(ctx, "Observation", id)
	if err != nil {
		return nil, err
	}

	var observation models.Observation
	if err := json.Unmarshal(respBody, &observation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal observation: %w", err)
	}

	return &observation, nil
}

// SearchObservations searches for observations
func (c *Client) SearchObservations(ctx context.Context, params SearchParams) ([]*models.Observation, error) {
	bundle, err := c.Search(ctx, "Observation", params)
	if err != nil {
		return nil, err
	}

	observations := make([]*models.Observation, 0, len(bundle.Entry))
	for _, entry := range bundle.Entry {
		var observation models.Observation
		if err := json.Unmarshal(entry.Resource, &observation); err != nil {
			continue
		}
		observations = append(observations, &observation)
	}

	return observations, nil
}

// GetCapabilityStatement retrieves the server's capability statement
func (c *Client) GetCapabilityStatement(ctx context.Context) (json.RawMessage, error) {
	return c.doRequest(ctx, "GET", "/metadata", nil)
}

// Everything retrieves all data for a patient ($everything operation)
func (c *Client) Everything(ctx context.Context, patientID string) (*Bundle, error) {
	path := fmt.Sprintf("/Patient/%s/$everything", patientID)
	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var bundle Bundle
	if err := json.Unmarshal(respBody, &bundle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal everything bundle: %w", err)
	}

	return &bundle, nil
}

// BearerTokenAuth implements bearer token authentication
type BearerTokenAuth struct {
	token string
}

// NewBearerTokenAuth creates a new bearer token auth provider
func NewBearerTokenAuth(token string) *BearerTokenAuth {
	return &BearerTokenAuth{token: token}
}

func (a *BearerTokenAuth) GetToken(ctx context.Context) (string, error) {
	return a.token, nil
}

func (a *BearerTokenAuth) TokenType() string {
	return "Bearer"
}

// OAuth2Auth implements OAuth2 authentication
type OAuth2Auth struct {
	tokenURL     string
	clientID     string
	clientSecret string
	scope        string
	token        string
	expiry       time.Time
	httpClient   *http.Client
}

// NewOAuth2Auth creates a new OAuth2 auth provider
func NewOAuth2Auth(tokenURL, clientID, clientSecret, scope string) *OAuth2Auth {
	return &OAuth2Auth{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		scope:        scope,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *OAuth2Auth) GetToken(ctx context.Context) (string, error) {
	if a.token != "" && time.Now().Before(a.expiry) {
		return a.token, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	if a.scope != "" {
		data.Set("scope", a.scope)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(a.clientID, a.clientSecret)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	a.token = tokenResp.AccessToken
	a.expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return a.token, nil
}

func (a *OAuth2Auth) TokenType() string {
	return "Bearer"
}
