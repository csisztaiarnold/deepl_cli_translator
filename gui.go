package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"io/ioutil"
	"sort"
	"strconv"
)

// The GUI version of the translator
func translateGUI() {
	a := app.New()
	w := a.NewWindow("DeepL Translator - " + Version)

	settings := loadSettings()

	apiKeyEntry := widget.NewEntry()
	apiKeyEntry.SetText(settings.APIKey)
	freeAPIEntry := widget.NewEntry()
	freeAPIEntry.SetText(settings.FreeAPI)
	sourceLangEntry := widget.NewEntry()
	sourceLangEntry.SetText(settings.SourceLang)
	targetLangEntry := widget.NewEntry()
	targetLangEntry.SetText(settings.TargetLang)
	warnCharacterLimitEntry := widget.NewEntry()
	warnCharacterLimitEntry.SetText(strconv.Itoa(settings.WarnCharacterLimit))
	maxRequestSizeEntry := widget.NewEntry()
	maxRequestSizeEntry.SetText(strconv.Itoa(settings.MaxRequestSize))

	saveConfiguration := func() {
		settings.APIKey = apiKeyEntry.Text
		settings.FreeAPI = freeAPIEntry.Text
		settings.SourceLang = sourceLangEntry.Text
		settings.TargetLang = targetLangEntry.Text
		settings.WarnCharacterLimit, _ = strconv.Atoi(warnCharacterLimitEntry.Text)
		settings.MaxRequestSize, _ = strconv.Atoi(maxRequestSizeEntry.Text)
		saveSettings(settings)
	}

	testAPIButton := widget.NewButton("Test API Key", func() {
		_, err := getRemainingCharacterLimit(settings.APIKey, settings.FreeAPI)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", "API Key is valid", w)
		}
	})

	inputTextArea := widget.NewMultiLineEntry()
	outputTextArea := widget.NewMultiLineEntry()

	remainingCharactersLabel := widget.NewLabel("Remaining Characters: N/A")

	updateRemainingCharacters := func() {
		remainingCharacters, err := getRemainingCharacterLimit(settings.APIKey, settings.FreeAPI)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		remainingCharactersLabel.SetText(fmt.Sprintf("Remaining Characters: %d", remainingCharacters))
	}

	translateButton := widget.NewButton("Translate", func() {
		inputText := inputTextArea.Text
		if len(inputText) > settings.WarnCharacterLimit {
			dialog.ShowConfirm("Warning", fmt.Sprintf("Your translated text exceeds the character limit of %d, do you want to continue?", settings.WarnCharacterLimit), func(confirmed bool) {
				if confirmed {
					proceedWithTranslation(inputText, settings, outputTextArea, w)
					updateRemainingCharacters()
				}
			}, w)
		} else {
			proceedWithTranslation(inputText, settings, outputTextArea, w)
			updateRemainingCharacters()
		}
	})

	filePicker := widget.NewButton("Select translation from a file", func() {
		dialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			defer reader.Close()
			data, err := ioutil.ReadAll(reader)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			inputTextArea.SetText(string(data))
		}, w)
		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".txt"}))
		dialog.Show()
	})

	translateFileButton := widget.NewButton("Save translation to a file", func() {
		translatedText, err := translateChunk(settings.APIKey, settings.SourceLang, settings.TargetLang, settings.FreeAPI, inputTextArea.Text)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				if writer == nil {
					return
				}
				defer writer.Close()
				writer.Write([]byte(translatedText))
				dialog.ShowInformation("Success", "File translated and saved", w)
			}, w).Show()
		}
	})

	checkInputText := func() {
		if inputTextArea.Text == "" {
			translateButton.Disable()
		} else {
			translateButton.Enable()
		}
	}

	checkOutputText := func() {
		if outputTextArea.Text == "" {
			translateFileButton.Disable()
		} else {
			translateFileButton.Enable()
		}
	}

	inputTextArea.OnChanged = func(text string) {
		checkInputText()
	}

	outputTextArea.OnChanged = func(text string) {
		checkOutputText()
	}

	checkInputText()
	checkOutputText()

	// Supported languages (https://developers.deepl.com/docs/resources/supported-languages)
	var supportedLanguages = map[string]string{
		"BG": "Bulgarian",
		"ZH": "Chinese",
		"CS": "Czech",
		"DA": "Danish",
		"NL": "Dutch",
		"EN": "English",
		"ET": "Estonian",
		"FI": "Finnish",
		"FR": "French",
		"DE": "German",
		"EL": "Greek",
		"HU": "Hungarian",
		"IT": "Italian",
		"JA": "Japanese",
		"LV": "Latvian",
		"LT": "Lithuanian",
		"PL": "Polish",
		"PT": "Portuguese",
		"RO": "Romanian",
		"RU": "Russian",
		"SK": "Slovak",
		"SL": "Slovenian",
		"ES": "Spanish",
		"SV": "Swedish",
	}

	freeAPIOptions := []string{"yes", "no"}

	freeAPISelect := widget.NewSelect(freeAPIOptions, func(selected string) {
		settings.FreeAPI = selected
	})

	freeAPISelect.SetSelected(settings.FreeAPI)

	var languageOptions []string
	for _, name := range supportedLanguages {
		languageOptions = append(languageOptions, name)
	}

	sort.Strings(languageOptions)

	settings = loadSettings()

	sourceLangSelect := widget.NewSelect(languageOptions, func(selected string) {
		for code, name := range supportedLanguages {
			if name == selected {
				settings.SourceLang = code
				sourceLangEntry.SetText(code)
				break
			}
		}
		saveConfiguration()
	})

	targetLangSelect := widget.NewSelect(languageOptions, func(selected string) {
		for code, name := range supportedLanguages {
			if name == selected {
				settings.TargetLang = code
				targetLangEntry.SetText(code)
				break
			}
		}
		saveConfiguration()
	})

	sourceLangSelect.SetSelected(supportedLanguages[settings.SourceLang])
	targetLangSelect.SetSelected(supportedLanguages[settings.TargetLang])

	apiKeyEntry.OnChanged = func(text string) {
		saveConfiguration()
	}

	freeAPIEntry.OnChanged = func(text string) {
		saveConfiguration()
	}

	sourceLangEntry.OnChanged = func(text string) {
		saveConfiguration()
	}

	targetLangEntry.OnChanged = func(text string) {
		saveConfiguration()
	}

	warnCharacterLimitEntry.OnChanged = func(text string) {
		saveConfiguration()
	}

	maxRequestSizeEntry.OnChanged = func(text string) {
		saveConfiguration()
	}

	form := container.NewVBox(
		container.NewVBox(
			widget.NewLabel("Your API Key"), apiKeyEntry,
			testAPIButton,
			container.NewGridWithColumns(2,
				widget.NewLabel("Use the free API plan"),
				widget.NewLabel("Warn Character Limit"),
			),
			container.NewGridWithColumns(2,
				freeAPISelect,
				warnCharacterLimitEntry,
			),
			widget.NewLabel("Max Request Size"), maxRequestSizeEntry,
		),
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabel("Source Language"),
				widget.NewLabel("Target Language"),
			),
			container.NewGridWithColumns(2,
				sourceLangSelect,
				targetLangSelect,
			),
			widget.NewLabel("Input Text"), inputTextArea,
			widget.NewLabel("Output Text"), outputTextArea,
		),
		container.NewGridWithColumns(3,
			translateButton, filePicker, translateFileButton,
		),
		remainingCharactersLabel,
	)

	marginContainer := container.New(layout.NewPaddedLayout(), form)

	w.SetContent(marginContainer)
	w.Resize(fyne.NewSize(800, 400))
	w.ShowAndRun()
}

func proceedWithTranslation(inputText string, settings Settings, outputTextArea *widget.Entry, w fyne.Window) {
	translatedText, err := translateChunk(settings.APIKey, settings.SourceLang, settings.TargetLang, settings.FreeAPI, inputText)
	if err != nil {
		dialog.ShowError(err, w)
	} else {
		outputTextArea.SetText(translatedText)
	}
}
