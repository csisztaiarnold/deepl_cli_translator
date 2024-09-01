package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"path/filepath"
	"strings"
)

// Help
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

// The CLI version of the translator
func translateCLI(inputFile, inputString, apiKey, freeAPI, sourceLang, targetLang, outputFile, output string) {

	if inputString == "" && inputFile == "" {
		fmt.Println("Either input_file or input_string must be provided")
		flag.Usage()
		return
	}

	if inputFile != "" {
		mimeType := mime.TypeByExtension(filepath.Ext(inputFile))
		if !strings.Contains(mimeType, "text") {
			fmt.Println("Input file must be a text file")
			return
		}
	}

	settings := Settings{
		WarnCharacterLimit: DefaultWarnCharacterLimit,
		MaxRequestSize:     DefaultMaxRequestSize,
	}

	settingsData, err := ioutil.ReadFile("settings.json")
	if err == nil {
		json.Unmarshal(settingsData, &settings)
	}

	if apiKey == "" {
		apiKey = settings.APIKey
	}

	freeAPIValue := flag.Lookup("free_api").Value.String()
	if freeAPIValue == "" {
		if settings.FreeAPI == "" {
			freeAPI = DefaultFreeAPI
		} else {
			freeAPI = settings.FreeAPI
		}
	} else {
		freeAPI = freeAPIValue
	}

	if sourceLang == "" {
		sourceLang = settings.SourceLang
	}

	if targetLang == "" {
		targetLang = settings.TargetLang
	}

	if apiKey == "" {
		fmt.Println("API key is required")
		flag.Usage()
		return
	}

	if sourceLang == "" || targetLang == "" {
		fmt.Println("source_lang and target_lang parameters are required")
		flag.Usage()
		return
	}

	var inputData []byte
	if inputString != "" {
		inputData = []byte(inputString)
	} else {
		inputData, err = ioutil.ReadFile(inputFile)
		if err != nil {
			fmt.Printf("Error reading input file: %v\n", err)
			return
		}
	}

	remainingCharacters, err := getRemainingCharacterLimit(apiKey, freeAPI)
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
		translatedChunk, err := translateChunk(apiKey, sourceLang, targetLang, freeAPI, string(inputData))
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
			translatedChunk, err := translateChunk(apiKey, sourceLang, targetLang, freeAPI, chunk)
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

	if output == "screen" {
		fmt.Println("Translated Text:")
		fmt.Println(translatedText)
	} else {
		if outputFile == "" {
			if inputString != "" {
				outputFile = sanitizeFilename(inputString)
			} else {
				ext := filepath.Ext(inputFile)
				outputFile = (inputFile)[:len(inputFile)-len(ext)] + ".translated" + ext
			}
		}

		if err := ioutil.WriteFile(outputFile, []byte(translatedText), 0644); err != nil {
			fmt.Printf("Error writing output file: %v\n", err)
			return
		}

		fmt.Println("Translation successful!")
	}

	remainingCharacters, err = getRemainingCharacterLimit(apiKey, freeAPI)
	if err != nil {
		fmt.Printf("Error getting remaining character limit: %v\n", err)
		return
	}
	fmt.Printf("Remaining character limit: %d\n", remainingCharacters)
}
