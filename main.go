package main

import (
	"flag"
)

func main() {
	runGUI := flag.Bool("gui", false, "Run the graphical user interface")
	inputFile := flag.String("input_file", "", "Path to the input file")
	inputString := flag.String("input_string", "", "Text to be translated")
	apiKey := flag.String("api_key", "", "DeepL API key")
	freeAPI := flag.String("free_api", "yes", "Use the free API endpoint")
	sourceLang := flag.String("source_lang", "", "Source language code")
	targetLang := flag.String("target_lang", "", "Target language code")
	outputFile := flag.String("output_file", "", "Path to the output file")
	output := flag.String("output", "file", "Output destination (file or screen)")
	flag.Parse()

	if *runGUI {
		translateGUI()
	} else {
		translateCLI(*inputFile, *inputString, *apiKey, *freeAPI, *sourceLang, *targetLang, *outputFile, *output)
	}
}
