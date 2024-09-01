package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
)

type Settings struct {
	APIKey             string `json:"api_key"`
	FreeAPI            string `json:"free_api"`
	SourceLang         string `json:"source_lang"`
	TargetLang         string `json:"target_lang"`
	WarnCharacterLimit int    `json:"warn_character_limit"`
	MaxRequestSize     int    `json:"max_request_size"`
}

type DeepLResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
}

type UsageResponse struct {
	CharacterCount int `json:"character_count"`
	CharacterLimit int `json:"character_limit"`
}

const (
	Version                   = "1.1.0"
	FreeAPIURL                = "https://api-free.deepl.com/v2/"
	PaidAPIURL                = "https://api.deepl.com/v2/"
	DefaultWarnCharacterLimit = 50000
	DefaultMaxRequestSize     = 8192
	DefaultFreeAPI            = "yes"
	ErrCreatingAPIRequest     = "error creating API request: %v"
	ErrMakingAPIRequest       = "error making API request: %v"
	ErrDecodingAPIResponse    = "error decoding API response: %v"
	APIErrorMsg               = "API error: %s\nStatus Code: %d\nResponse Body: %s\nPlease make sure your API key is working."
)

// Load settings from settings.json
func loadSettings() Settings {
	var settings Settings
	data, err := ioutil.ReadFile("settings.json")
	if err == nil {
		json.Unmarshal(data, &settings)
	}
	return settings
}

// Save settings to settings.json
func saveSettings(settings Settings) {
	data, _ := json.MarshalIndent(settings, "", "  ")
	ioutil.WriteFile("settings.json", data, 0644)
}

// Makes an API request and returns the response.
func makeAPIRequest(apiURL, apiKey string, data url.Values) (*http.Response, error) {
	data.Set("auth_key", apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf(ErrCreatingAPIRequest, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(ErrMakingAPIRequest, err)
	}
	return resp, nil
}

// Get the remaining character limit from the API
func getRemainingCharacterLimit(apiKey string, freeAPI string) (int, error) {
	apiURL := getAPIURL("usage", freeAPI)
	data := url.Values{}

	resp, err := makeAPIRequest(apiURL, apiKey, data)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		re := regexp.MustCompile(`<.*?>`)
		cleanedBody := re.ReplaceAllString(string(body), "")
		return 0, fmt.Errorf(APIErrorMsg, cleanedBody, resp.StatusCode, string(body))
	}

	var usageResp UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		return 0, fmt.Errorf(ErrDecodingAPIResponse, err)
	}

	return usageResp.CharacterLimit - usageResp.CharacterCount, nil
}

// Sends API requests in chunks in order to avoid timeouts
func translateChunk(apiKey, sourceLang, targetLang string, freeAPI string, chunk string) (string, error) {
	apiURL := getAPIURL("translate", freeAPI)
	data := url.Values{
		"text":        {chunk},
		"source_lang": {sourceLang},
		"target_lang": {targetLang},
	}

	resp, err := makeAPIRequest(apiURL, apiKey, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		re := regexp.MustCompile(`<.*?>`)
		cleanedBody := re.ReplaceAllString(string(body), "")
		return "", fmt.Errorf(APIErrorMsg, cleanedBody, resp.StatusCode, string(body))
	}

	var deeplResp DeepLResponse
	if err := json.NewDecoder(resp.Body).Decode(&deeplResp); err != nil {
		return "", fmt.Errorf(ErrDecodingAPIResponse, err)
	}

	return deeplResp.Translations[0].Text, nil
}

// Generates the API URL based on the endpoint and whether the free or the paid API plan is specified.
func getAPIURL(endpoint string, freeAPI string) string {
	if freeAPI == DefaultFreeAPI {
		return FreeAPIURL + endpoint
	}
	return PaidAPIURL + endpoint
}

// Creates a filename from the input string
func sanitizeFilename(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z]`)
	sanitized := re.ReplaceAllString(input, "_")
	if len(sanitized) > 15 {
		sanitized = sanitized[:15]
	}
	return sanitized + ".txt"
}
