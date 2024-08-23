package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const Version = "1.0.0"

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

func usage() {
	fmt.Println("Usage: translator -input_string <text> -input_file <path> -output_file <path> -api_key <key> -source_lang <lang_code> -target_lang <lang_code> -free_api <yes|no> -output <file|screen>")
	fmt.Println("Parameters:")
	fmt.Println("  -input_file   Path to the input file")
	fmt.Println("  -api_key      DeepL API key")
	fmt.Println("  -free_api     Use the free API endpoint (default: yes)")
	fmt.Println("  -source_lang  Source language code")
	fmt.Println("  -target_lang  Target language code")
	fmt.Println("  -output_file  Path to the output file")
	fmt.Println("  -output       Output destination (file or screen, default: file)")
	fmt.Println("  -input_string Text to be translated")
}

// Takes an input string and returns a sanitized version of it. If the sanitized
// string is longer than 10 characters, it truncates it to 10 characters and appends
// ".txt" to the end in order to create a valid filename.
func sanitizeFilename(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z]`)
	sanitized := re.ReplaceAllString(input, "_")
	if len(sanitized) > 15 {
		sanitized = sanitized[:15]
	}
	return sanitized + ".txt"
}

// Generates the API URL based on the endpoint and whether the free or the paid API plan is specified.
func getAPIURL(endpoint string, freeAPI string) string {
	if freeAPI == "yes" {
		return "https://api-free.deepl.com/v2/" + endpoint
	}
	return "https://api.deepl.com/v2/" + endpoint
}

// Makes an API request and returns the response.
func makeAPIRequest(apiURL, apiKey string, data url.Values) (*http.Response, error) {
	data.Set("auth_key", apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating API request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making API request: %v", err)
	}
	return resp, nil
}

// Gets the remaining character limit for the API.
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
		return 0, fmt.Errorf("API error: %s\nStatus Code: %d\nResponse Body: %s\nPlease make sure your API key is working.", cleanedBody, resp.StatusCode, string(body))
	}

	var usageResp UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		return 0, fmt.Errorf("error decoding API response: %v", err)
	}

	return usageResp.CharacterLimit - usageResp.CharacterCount, nil
}

// Translates a chunk of text in a size that is within the character limit.
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
		return "", fmt.Errorf("API error: %s\nStatus Code: %d\nResponse Body: %s\nPlease make sure your API key is working.", cleanedBody, resp.StatusCode, string(body))
	}

	var deeplResp DeepLResponse
	if err := json.NewDecoder(resp.Body).Decode(&deeplResp); err != nil {
		return "", fmt.Errorf("error decoding API response: %v", err)
	}

	return deeplResp.Translations[0].Text, nil
}

func main() {
	flag.Usage = usage

	fmt.Println("DeepL CLI Translator ", Version)

	// The flags.
	inputFile := flag.String("input_file", "", "Path to the input file")
	inputString := flag.String("input_string", "", "Text to be translated")
	apiKey := flag.String("api_key", "", "DeepL API key")
	freeAPI := flag.String("free_api", "", "Use the free API endpoint")
	sourceLang := flag.String("source_lang", "", "Source language code")
	targetLang := flag.String("target_lang", "", "Target language code")
	outputFile := flag.String("output_file", "", "Path to the output file")
	output := flag.String("output", "file", "Output destination (file or screen)")
	flag.Parse()

	if *inputString == "" && *inputFile == "" {
		fmt.Println("Either input_file or input_string must be provided")
		flag.Usage()
		return
	}

	// Is the file textual?
	if *inputFile != "" {
		mimeType := mime.TypeByExtension(filepath.Ext(*inputFile))
		if !strings.Contains(mimeType, "text") {
			fmt.Println("Input file must be a text file")
			return
		}
	}

	settings := Settings{
		WarnCharacterLimit: 50000,
		MaxRequestSize:     8192,
	}
	settingsData, err := ioutil.ReadFile("settings.json")
	if err == nil {
		json.Unmarshal(settingsData, &settings)
	}

	if *apiKey == "" {
		*apiKey = settings.APIKey
	}

	// If the free_api flag is not set, use the settings value.
	// If there's no default settings value, use "yes".
	freeAPIValue := flag.Lookup("free_api").Value.String()
	if freeAPIValue == "" {
		if settings.FreeAPI == "" {
			*freeAPI = "yes"
		} else {
			*freeAPI = settings.FreeAPI
		}
	} else {
		*freeAPI = freeAPIValue
	}

	if *sourceLang == "" {
		*sourceLang = settings.SourceLang
	}

	if *targetLang == "" {
		*targetLang = settings.TargetLang
	}

	if *apiKey == "" {
		fmt.Println("API key is required")
		flag.Usage()
		return
	}

	if *sourceLang == "" || *targetLang == "" {
		fmt.Println("source_lang and target_lang parameters are required")
		flag.Usage()
		return
	}

	var inputData []byte
	if *inputString != "" {
		inputData = []byte(*inputString)
	} else {
		inputData, err = ioutil.ReadFile(*inputFile)
		if err != nil {
			fmt.Printf("Error reading input file: %v\n", err)
			return
		}
	}

	remainingCharacters, err := getRemainingCharacterLimit(*apiKey, *freeAPI)
	if err != nil {
		fmt.Printf("Error getting remaining character limit: %v\n", err)
		return
	}

	if len(inputData) > remainingCharacters {
		fmt.Printf("Input data exceeds the remaining character limit of %d characters\n", remainingCharacters)
		return
	}

	var translatedTextBuilder strings.Builder
	if len(inputData) <= settings.MaxRequestSize {
		translatedChunk, err := translateChunk(*apiKey, *sourceLang, *targetLang, *freeAPI, string(inputData))
		if err != nil {
			fmt.Printf("Error translating text: %v\n", err)
			return
		}
		translatedTextBuilder.WriteString(translatedChunk)
	} else {
		totalChunks := (len(inputData) + settings.MaxRequestSize - 1) / settings.MaxRequestSize
		for i := 0; i < totalChunks; i++ {
			start := i * settings.MaxRequestSize
			end := start + settings.MaxRequestSize
			if end > len(inputData) {
				end = len(inputData)
			}
			chunk := string(inputData[start:end])
			translatedChunk, err := translateChunk(*apiKey, *sourceLang, *targetLang, *freeAPI, chunk)
			if err != nil {
				fmt.Printf("Error translating chunk: %v\n", err)
				return
			}
			translatedTextBuilder.WriteString(translatedChunk)
			fmt.Print(".")
		}
		fmt.Println()
	}

	translatedText := translatedTextBuilder.String()

	if *output == "screen" {
		fmt.Println("Translated Text:")
		fmt.Println(translatedText)
	} else {
		if *outputFile == "" {
			if *inputString != "" {
				*outputFile = sanitizeFilename(*inputString)
			} else {
				ext := filepath.Ext(*inputFile)
				*outputFile = (*inputFile)[:len(*inputFile)-len(ext)] + ".translated" + ext
			}
		}

		if err := ioutil.WriteFile(*outputFile, []byte(translatedText), 0644); err != nil {
			fmt.Printf("Error writing output file: %v\n", err)
			return
		}

		fmt.Println("Translation successful!")
	}

	remainingCharacters, err = getRemainingCharacterLimit(*apiKey, *freeAPI)
	if err != nil {
		fmt.Printf("Error getting remaining character limit: %v\n", err)
		return
	}
	fmt.Printf("Remaining character limit: %d\n", remainingCharacters)
}
