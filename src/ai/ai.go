/*
		use http and not api libraries
	   		size application: 17Mo to 7,4Mo !! (for only gemini)
*/
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	iaServices = []string{"GEMINI", "MISTRAL", "OPENAI"}
)

type (
	IaiBase interface {
		Set()
		Ask(prompt string) string
	}

	AiBase struct {
		url   string
		Model string
		key   string
	}
	IaiGemini  struct{ AiBase }
	IaiMistral struct{ AiBase }
	IaiOpenai  struct{ AiBase }
)

func (a *AiBase) Ask(prompt string) string {
	return ""
}

// -------------------------------------------------

func MakeAi(key string) (ai IaiBase) {

	var test = func(key string) IaiBase {
		k := os.Getenv(key)
		if k != "" {
			if strings.Contains(key, "GEMINI") {
				ai = &IaiGemini{}
				ai.Set()
				return ai
			}
			if strings.Contains(key, "MISTRAL") {
				ai = &IaiMistral{}
				ai.Set()
				return ai
			}
		}
		return nil
	}
	key = strings.ToUpper(key)
	ai = nil
	if key != "" {
		if k := os.Getenv(key); k != "" {
			return test(key)
		}
	}
	for _, service := range iaServices {
		t := test(service + "_API_KEY")
		if t != nil {
			return t
		}
	}
	return ai
}

// -------------------------------------------------

func (a *IaiGemini) Set() {
	a.url = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s"
	a.Model = "gemini-2.0-flash"
	a.key = os.Getenv("GEMINI_API_KEY")
}

func (a IaiGemini) Ask(prompt string) string {
	//fmt.Println("#", a.Model)

	type (
		Part struct {
			Text string `json:"text"`
		}

		Content struct {
			Parts []Part `json:"parts"`
			Role  string `json:"role"`
		}

		TokensDetails struct {
			Modality   string `json:"modality"`
			TokenCount int    `json:"tokenCount"`
		}

		UsageMetadata struct {
			PromptTokenCount        int             `json:"promptTokenCount"`
			CandidatesTokenCount    int             `json:"candidatesTokenCount"`
			TotalTokenCount         int             `json:"totalTokenCount"`
			PromptTokensDetails     []TokensDetails `json:"promptTokensDetails"`
			CandidatesTokensDetails []TokensDetails `json:"candidatesTokensDetails"`
		}

		Candidate struct {
			Content      Content `json:"content"`
			FinishReason string  `json:"finishReason"`
			AvgLogprobs  float64 `json:"avgLogprobs"`
		}

		GeminiResponse struct {
			Candidates    []Candidate   `json:"candidates"`
			UsageMetadata UsageMetadata `json:"usageMetadata"`
			ModelVersion  string        `json:"modelVersion"`
		}
	)

	url := fmt.Sprintf(a.url, a.key)

	ask := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
				},
			},
		},
	}
	askBytes, _ := json.Marshal(ask)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(askBytes))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var response GeminiResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return ""
	}

	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		return response.Candidates[0].Content.Parts[0].Text
	}

	return ""
}

// -------------------------------------------------

func (a *IaiMistral) Set() {
	a.url = "https://api.mistral.ai/v1/chat/completions"
	a.Model = "mistral-large-latest"
	a.key = os.Getenv("MISTRAL_API_KEY")
}

func (a IaiMistral) Ask(prompt string) string {
	//fmt.Println("#", a.Model)

	type (
		ChatMessage struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		MistralRequest struct {
			Model    string        `json:"model"`
			Messages []ChatMessage `json:"messages"`
		}

		Choice struct {
			Index        int         `json:"index"`
			Message      ChatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}

		UsageInfo struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}

		// Définition de la structure Go pour la réponse Mistral AI
		MistralResponse struct {
			Id      string    `json:"id"`
			Object  string    `json:"object"`
			Created int64     `json:"created"`
			Model   string    `json:"model"`
			Choices []Choice  `json:"choices"`
			Usage   UsageInfo `json:"usage"`
		}
	)

	ask := MistralRequest{
		Model: a.Model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
			{Role: "system", Content: "response on one line"},
		},
	}
	askBytes, _ := json.Marshal(ask)

	req, err := http.NewRequest("POST", a.url, bytes.NewBuffer(askBytes))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.key))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var response MistralResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return ""
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content
	}

	return ""
}

// -------------------------------------------------

func GetAskPackage(pkgname, repo string) string {

	req := `
		Use this lang ` + os.Getenv("LANG") + ` for the response.
		Response not in markdown but in simple text for console, text/plain.

		Informations on a pacman package in Manjaro linux, or in archlinux.
		This package is in repository : ` + repo + `

		Informations on utility of this manjaro package : ` + pkgname + `
		If this package have an application, can you add a descriptif of this app ? Maximum 10 lines.
	`
	return req
}
