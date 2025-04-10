package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type (
	AiLmm struct {
		Ctx    context.Context
		Client *genai.Client
		Model  *genai.GenerativeModel
	}
)

func (a *AiLmm) Close() {
	if a.Client != nil {
		a.Client.Close()
	}
}

func (a *AiLmm) Init(ctx context.Context) error {
	if a.Client != nil {
		return nil
	}
	a.Ctx = ctx
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return err
	}
	a.Client = client
	a.Model = client.GenerativeModel("gemini-2.0-flash")
	return nil
}

func (a AiLmm) Ask(prompt string) (string, error) {
	if a.Client == nil || a.Model == nil {
		return "", nil
	}
	resp, err := a.Model.GenerateContent(a.Ctx, genai.Text(prompt))
	if err != nil {
		return "", nil
	}
	return fmt.Sprint(resp.Candidates[0].Content.Parts[0]), nil
}

func (a AiLmm) AskPackage(pkgname, repo string) string {

	req := `
		Use this lang ` + os.Getenv("LANG") + ` for the response.
		Response not in markdown but in simple text for console, text/plain.

		Informations on a pacman package in Manjaro linux, or in archlinux.
		This package is in repository : ` + repo + `
		
		Informations on utility of this manjaro package : ` + pkgname + `
		If this package have an application, can you add a descriptif of this app ? Maximum 10 lines.
	`
	response, err := a.Ask(req)
	if err != nil {
		return ""
	}
	return response
}
