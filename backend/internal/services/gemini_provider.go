package services

import (
	"context"
	"errors"

	"google.golang.org/genai"
)

type GeminiProvider struct {
	client *genai.Client
	model  string
}

func NewGeminiProvider(apiKey string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, errors.New("gemini API key is missing")
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &GeminiProvider{
		client: client,
		model:  "gemini-3.6-flash",
	}, nil
}

func (p *GeminiProvider) Analyze(
	request AIRequest,
) (*DiagnosisResult, error) {

	if request.Category == "" &&
		request.Description == "" &&
		len(request.Image) == 0 &&
		len(request.Audio) == 0 &&
		len(request.Video) == 0 {
		return nil, errors.New("no information was provided")
	}

	prompt := `You are Agro-Shield AI, a friendly agricultural assistant.

Your users are everyday farmers. Explain everything in very simple English.
Use short sentences and common words.

Do not use difficult scientific words unless necessary.
If you use a difficult word, explain its meaning immediately.

Do not use Markdown.
Do not use symbols such as **, ##, ###, or horizontal lines.
Do not write a long introduction.
Do not repeat the farmer's question.

Use this exact format:

Problem:
Explain the likely problem in one or two simple sentences.

What I noticed:
Give up to three simple signs that support your answer.

What you should do:
Give three to five clear actions the farmer can take.

Important:
Say clearly when you are not certain.
Do not claim that your answer is a guaranteed diagnosis.
Tell the farmer to contact a local agricultural officer or veterinarian
if the problem may be serious.

Category:
` + request.Category + `

Farmer's description:
` + request.Description + `

Analyze the attached image, audio, or video if one was provided.
Keep the complete answer below 250 words.`

	parts := []*genai.Part{
		{Text: prompt},
	}

	if len(request.Image) > 0 {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     request.Image,
				MIMEType: request.ImageType,
			},
		})
	}

	if len(request.Audio) > 0 {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     request.Audio,
				MIMEType: request.AudioType,
			},
		})
	}

	if len(request.Video) > 0 {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     request.Video,
				MIMEType: request.VideoType,
			},
		})
	}

	contents := []*genai.Content{
		{
			Parts: parts,
		},
	}

	result, err := p.client.Models.GenerateContent(
		context.Background(),
		p.model,
		contents,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if result == nil || len(result.Candidates) == 0 {
		return nil, errors.New("gemini returned no response")
	}

	for _, candidate := range result.Candidates {
		if candidate.Content == nil {
			continue
		}

		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				return &DiagnosisResult{
					Result: part.Text,
				}, nil
			}
		}
	}

	return nil, errors.New("gemini returned no diagnosis text")
}
