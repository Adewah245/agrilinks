package services

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"backend/internal/models"

	"google.golang.org/genai"
)

// GeminiProvider connects Agro-Shield to Google's Gemini AI service.
//
// It implements the AIProvider interface and is responsible for
// sending agricultural diagnosis requests to Gemini and converting
// Gemini's response into an Agro-Shield diagnosis result.
type GeminiProvider struct {
	client *genai.Client
	model  string
}

// NewGeminiProvider creates a new GeminiProvider using the supplied
// Gemini API key.
//
// The provider uses Google's Gemini API backend and returns an error
// when the API key is missing or the Gemini client cannot be created.
func NewGeminiProvider(apiKey string, modelName ...string) (*GeminiProvider, error) {
	apiKey = strings.TrimSpace(apiKey)

	// Check that the Gemini API key was provided.
	if apiKey == "" {
		return nil, errors.New("gemini API key is missing")
	}
	if len(modelName) != 1 || strings.TrimSpace(modelName[0]) == "" {
		return nil, errors.New("Gemini model is required")
	}

	// Create the Gemini client.
	client, err := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		},
	)
	if err != nil {
		return nil, err
	}

	// Return the Gemini provider Agro-Shield will use.
	return &GeminiProvider{
		client: client,

		// The Gemini model Agro-Shield is currently configured to use.
		model: strings.TrimSpace(modelName[0]),
	}, nil
}

// Analyze sends an agricultural AI request to Gemini.
//
// The request may contain:
//
//   - Text
//   - Image
//   - Audio
//   - Video
//
// The method validates the request, prepares any image data,
// sends the request to Gemini, retries temporary service errors,
// and returns the generated diagnosis text.
func (p *GeminiProvider) Analyze(
	request models.AIRequest,
) (*models.DiagnosisResult, error) {

	// ========================================================
	// CHECK PROVIDER
	// ========================================================

	if p == nil || p.client == nil {
		return nil, errors.New("gemini provider is not initialized")
	}

	if strings.TrimSpace(p.model) == "" {
		return nil, errors.New("gemini model is not configured")
	}

	// ========================================================
	// CHECK USER INPUT
	// Make sure the user actually sent something.
	// ========================================================

	if strings.TrimSpace(request.Category) == "" &&
		strings.TrimSpace(request.Description) == "" &&
		len(request.Image) == 0 &&
		len(request.Audio) == 0 &&
		len(request.Video) == 0 {

		return nil, errors.New("no information was provided")
	}

	// ========================================================
	// AGRO-SHIELD AI INSTRUCTIONS
	// This tells Gemini how Agro-Shield should communicate.
	// ========================================================

	prompt := `You are Agro-Shield AI, a friendly farming assistant for everyday farmers in Nigeria.

Talk to the farmer the same simple way a good mentor would talk to a friend.

Use very simple English. Use common Nigerian English that farmers can easily understand.

Do not use big grammar or difficult words.

Keep your sentences short and clear.

If you must use a difficult farming or medical word, explain it immediately in simple words.

Talk naturally and respectfully. Do not sound like a textbook, professor, or robot.

The farmer may not know much about technology, science, or farming terms, so explain things step by step.

Do not assume the farmer already understands the problem.

Do not use Markdown.

Do not use symbols such as **, ##, ###, bullet symbols, or horizontal lines.

Do not write a long introduction.

Do not repeat the farmer's question.

Do not give too much information at once.

Use this exact format:

Problem:

Tell the farmer what may be wrong in one or two simple sentences.

What you will commonly see are:

Give up to five simple signs the farmer may notice.

What you should do:

Give three to five simple things the farmer can do.

Important:

Be honest when you are not sure.

Never say that your answer is a guaranteed diagnosis.

If the problem looks serious, tell the farmer to contact a local agricultural officer, extension worker, or veterinarian.

Always focus on giving advice the farmer can understand and act on.

Your goal is not to impress the farmer with big words.

Your goal is to help the farmer understand the problem and know what to do next.

Category:
` + strings.TrimSpace(request.Category) + `

User's description:
` + strings.TrimSpace(request.Description) + `

Analyze the attached image, audio, or video if one was provided.
Keep the complete answer below 250 words.`

	// ========================================================
	// CREATE AI REQUEST PARTS
	// Start with the text instructions.
	// ========================================================

	parts := []*genai.Part{
		{
			Text: prompt,
		},
	}

	// ========================================================
	// ADD IMAGE
	// Process the image before sending it to Gemini.
	//
	// Large phone images can make AI analysis slower.
	// We resize and compress the image first.
	// ========================================================

	if len(request.Image) > 0 {
		processedImage, err := prepareImage(request.Image)
		if err != nil {
			log.Printf("image processing failed: %v", err)
			return nil, errors.New("could not process image")
		}

		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     processedImage,
				MIMEType: "image/jpeg",
			},
		})
	}

	// ========================================================
	// ADD AUDIO
	// If the user sent an audio recording, attach it.
	// ========================================================

	if len(request.Audio) > 0 {
		audioType := strings.TrimSpace(request.AudioType)

		if audioType == "" {
			return nil, errors.New("audio type is missing")
		}

		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     request.Audio,
				MIMEType: audioType,
			},
		})
	}

	// ========================================================
	// ADD VIDEO
	// If the user sent a video, attach it.
	// ========================================================

	if len(request.Video) > 0 {
		videoType := strings.TrimSpace(request.VideoType)

		if videoType == "" {
			return nil, errors.New("video type is missing")
		}

		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     request.Video,
				MIMEType: videoType,
			},
		})
	}

	// ========================================================
	// BUILD GEMINI CONTENT
	// Put all the information together before sending it.
	// ========================================================

	contents := []*genai.Content{
		{
			Parts: parts,
		},
	}

	// ========================================================
	// BUILD GENERATION CONFIGURATION
	//
	// Thinking is disabled by default because Agro-Shield's
	// diagnosis response should be reasonably fast.
	//
	// When Thinking is explicitly requested, we allow a
	// limited thinking budget.
	// ========================================================

	thinkingBudget := int32(0)

	if request.Thinking {
		thinkingBudget = 1024
	}

	config := &genai.GenerateContentConfig{
		MaxOutputTokens: 300,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: genai.Ptr(thinkingBudget),
		},
	}

	// ========================================================
	// SEND REQUEST TO GEMINI
	//
	// The request has:
	//
	// - A 60-second timeout.
	// - Up to 3 attempts.
	// - Retry handling for temporary Gemini failures.
	// ========================================================

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	var result *genai.GenerateContentResponse
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		result, err = p.client.Models.GenerateContent(
			ctx,
			p.model,
			contents,
			config,
		)

		if err == nil || !isTemporaryGeminiError(err) {
			break
		}

		log.Printf(
			"temporary Gemini error on attempt %d: %v",
			attempt+1,
			err,
		)

		if attempt >= 2 {
			break
		}

		delay := time.Duration(attempt+1) * time.Second

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	// ========================================================
	// HANDLE GEMINI ERROR
	// ========================================================

	if err != nil {
		log.Printf(
			"Gemini analysis failed: %v",
			err,
		)

		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(ctx.Err(), context.DeadlineExceeded) {

			return nil, errors.New(
				"the AI service took too long to respond. Please try again",
			)
		}

		if isTemporaryGeminiError(err) {
			return nil, errors.New(
				"the AI service is busy right now. Please wait a moment and try again",
			)
		}

		return nil, errors.New(
			"the AI service could not complete the analysis. Please try again",
		)
	}

	// ========================================================
	// CHECK GEMINI RESPONSE
	// Make sure Gemini actually returned something.
	// ========================================================

	if result == nil {
		return nil, errors.New(
			"gemini returned no response",
		)
	}

	if len(result.Candidates) == 0 {
		return nil, errors.New(
			"gemini returned no response",
		)
	}

	// ========================================================
	// GET THE AI TEXT
	// Find the usable text inside Gemini's response.
	// ========================================================

	for _, candidate := range result.Candidates {

		if candidate == nil || candidate.Content == nil {
			continue
		}

		for _, part := range candidate.Content.Parts {

			if part == nil {
				continue
			}

			if strings.TrimSpace(part.Text) != "" {
				return &models.DiagnosisResult{
					Result: strings.TrimSpace(part.Text),
				}, nil
			}
		}
	}

	// ========================================================
	// NO DIAGNOSIS FOUND
	// Gemini responded, but there was no usable text.
	// ========================================================

	return nil, errors.New(
		"gemini returned no diagnosis text",
	)
}

// isTemporaryGeminiError determines whether a Gemini error
// represents a temporary service problem that can reasonably
// be retried.
func isTemporaryGeminiError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())

	return strings.Contains(message, "503") ||
		strings.Contains(message, "unavailable") ||
		strings.Contains(message, "high demand") ||
		strings.Contains(message, "resource exhausted") ||
		strings.Contains(message, "429")
}
