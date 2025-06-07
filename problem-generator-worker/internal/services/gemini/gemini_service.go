package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/your-username/edumint/problem-generator-worker/internal/models"
	"google.golang.org/api/option"
)

type Service struct {
	extractionClient *genai.GenerativeModel
	generationClient *genai.GenerativeModel
}

const (
	defaultExtractionModel = "gemini-2.5-flash-preview-05-20"
	defaultGenerationModel = "gemini-2.5-flash-preview-05-20"
)

var invalidEscapeRegex = regexp.MustCompile(`\\([^"\\/bfnrtu])`)

func NewService() (*Service, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}
	extractionModelName := os.Getenv("GEMINI_EXTRACTION_MODEL")
	if extractionModelName == "" {
		log.Printf("Warning: GEMINI_EXTRACTION_MODEL not set, using default '%s'", defaultExtractionModel)
		extractionModelName = defaultExtractionModel
	}
	generationModelName := os.Getenv("GEMINI_GENERATION_MODEL")
	if generationModelName == "" {
		log.Printf("Warning: GEMINI_GENERATION_MODEL not set, using default '%s'", defaultGenerationModel)
		generationModelName = defaultGenerationModel
	}
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	extractionModel := client.GenerativeModel(extractionModelName)
	extractionModel.ResponseMIMEType = "application/json"

	generationModel := client.GenerativeModel(generationModelName)
	generationModel.ResponseMIMEType = "application/json"

	log.Printf("GeminiService initialized with Extraction Model: %s and Generation Model: %s (JSON Mode ENABLED)", extractionModelName, generationModelName)
	return &Service{extractionClient: extractionModel, generationClient: generationModel}, nil
}

func (s *Service) ExtractStructure(parts ...genai.Part) (*models.ProblemStructure, *genai.UsageMetadata, error) {
	if s.extractionClient == nil {
		return nil, nil, fmt.Errorf("gemini extraction client not initialized")
	}
	promptParts := []genai.Part{genai.Text(structureExtractionPromptTemplate)}
	promptParts = append(promptParts, parts...)

	ctx := context.Background()
	resp, err := s.extractionClient.GenerateContent(ctx, promptParts...)
	if err != nil {
		return nil, nil, fmt.Errorf("extraction model failed to generate content: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, resp.UsageMetadata, fmt.Errorf("no content from Gemini for structure extraction")
	}

	jsonOutput, err := parseAndCleanAndFixJSONResponse(resp)
	if err != nil {
		return nil, resp.UsageMetadata, err
	}

	var problemStructure models.ProblemStructure
	if err := json.Unmarshal([]byte(jsonOutput), &problemStructure); err != nil {
		return nil, resp.UsageMetadata, fmt.Errorf("failed to unmarshal final JSON (structure): %w. Final JSON string: %s", err, jsonOutput)
	}
	return &problemStructure, resp.UsageMetadata, nil
}

func (s *Service) GenerateProblem(problemStructure *models.ProblemStructure) (*models.GeneratedData, *genai.UsageMetadata, error) {
	if s.generationClient == nil {
		return nil, nil, fmt.Errorf("gemini generation client not initialized")
	}
	structureBytes, err := json.MarshalIndent(problemStructure, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal problem structure: %w", err)
	}

	prompt := fmt.Sprintf(problemAndAnswerGenerationPromptTemplate, string(structureBytes))

	ctx := context.Background()
	resp, err := s.generationClient.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, nil, fmt.Errorf("generation model failed to generate content: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, resp.UsageMetadata, fmt.Errorf("no content from Gemini for problem generation")
	}

	jsonOutput, err := parseAndCleanAndFixJSONResponse(resp)
	if err != nil {
		return nil, resp.UsageMetadata, err
	}

	var generatedOutput models.GeneratedData
	if err := json.Unmarshal([]byte(jsonOutput), &generatedOutput); err != nil {
		return nil, resp.UsageMetadata, fmt.Errorf("failed to unmarshal final JSON (problem/answer): %w. Final JSON string: %s", err, jsonOutput)
	}
	return &generatedOutput, resp.UsageMetadata, nil
}

// parseAndCleanAndFixJSONResponse safely extracts and fixes malformed JSON.
func parseAndCleanAndFixJSONResponse(resp *genai.GenerateContentResponse) (string, error) {
	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			sb.WriteString(string(txt))
		}
	}
	rawResponse := sb.String()

	if rawResponse == "" {
		return "", fmt.Errorf("AI response was empty")
	}

	if strings.Contains(rawResponse, "```") {
		re := regexp.MustCompile("(?s)```json(.*)```")
		matches := re.FindStringSubmatch(rawResponse)
		if len(matches) >= 2 {
			rawResponse = matches[1]
		}
	}

	trimmed := strings.TrimSpace(rawResponse)

	// Step 1: Try raw JSON
	if isValidJSON(trimmed) {
		return trimmed, nil
	}

	// Step 2: Apply fix
	sanitized := sanitizeJSONString(trimmed)

	if isValidJSON(sanitized) {
		return sanitized, nil
	}

	return "", fmt.Errorf("failed to parse valid JSON even after sanitization: %s", sanitized)
}

func sanitizeJSONString(s string) string {
	return invalidEscapeRegex.ReplaceAllString(s, `\\$1`)
}

func isValidJSON(input string) bool {
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.DisallowUnknownFields()
	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return false
	}
	return true
}

// プロンプトを厳格化し、AIがタスクを誤解したり、余計なテキストを出力したりするのを防ぎます。
const structureExtractionPromptTemplate = `あなたのタスクは、以下の「入力テキスト」を分析し、指定されたJSON形式で出力することです。
**最重要ルール: あなたの応答は、JSONオブジェクト自体で開始し、終了する必要があります。JSONの前後に、いかなる説明、前置き、言い訳、その他のテキストも絶対に追加しないでください。**

**ステップ1：入力タイプの判断**
まず、入力テキストが**「試験問題形式」**か、それとも**「講義ノートや教科書のような学習資料形式」**かを判断してください。

**ステップ2：タイプに応じた処理の実行**

**【入力が「試験問題形式」の場合】**
テキストに書かれている問題の構造を**そのまま抽出**し、JSONにマッピングしてください。
- ` + "`exam_meta`" + `: 試験のタイトルや設定を抽出します。
- ` + "`major_sections`" + `: 大問を抽出します。
- ` + "`sub_questions`" + `: 各大問を小問ごとに区切ってください。` + "`question_index`" + `には問題番号をそのまま使います。

**【入力が「講義ノートや学習資料形式」の場合】**
その資料から**新しい試験問題を作成するための設計図**として、JSONを**提案・生成**してください。
- ` + "`exam_meta.exam_title`" + `: 資料全体を代表する架空の試験タイトル（例: 「〇〇学 期末試験設計案」）を設定してください。
- ` + "`major_sections`" + `: 資料内容を論理的なセクションに分割し、` + "`section_title`" + `を設定してください。
- ` + "`sub_questions`" + `: 各セクションで問うべき具体的な小問を、キーワードレベルで設定してください。

**共通ルール:**
- **問題を解いたり、JSON以外のテキストを出力したりしないでください。**
- 出力は**必ず**指定されたJSONオブジェクト形式でなければなりません。

**JSON出力スキーマ:**
{
  "exam_meta": { "exam_title": "string", "exam_duration": "integer (minutes)", "open_book": "boolean", "allowed_materials": ["string"], "question_format_is_latex": "boolean", "answer_format_is_latex": "boolean" },
  "structure": {
    "major_sections": [
      { "section_index": "string", "section_title": "string", "sub_questions": [ { "question_index": "string", "topic": "string", "keywords": ["string"], "difficulty": "string" } ] }
    ]
  }
}`

const problemAndAnswerGenerationPromptTemplate = `以下の問題構造情報に基づいて、新しい大学レベルの試験問題とその解答のセットを生成してください。

**最重要ルール:**
- **あなたの応答は、JSONオブジェクト自体で開始し、終了する必要があります。JSONの前後に、いかなる説明、前置き、その他のテキストも絶対に追加しないでください。**
- **絶対に、LaTeXのプリアンブルやドキュメント環境コマンド（例: \documentclass, \usepackage, \begin{document}, \end{document}）を含めないでください。**
- ` + "`question_text`と`answer_text`" + `の値は、Markdown形式のテキスト本体のみにしてください。
- 数式はインライン($...$)またはディスプレイ($$...$$)形式で記述してください。
- **注意: 文字列中にバックスラッシュ（\）を使用する場合、JSONの構文規則に従い、必ず\\と二重にエスケープしてください。**


**JSON出力スキーマ:**
{
  "exam_meta": { "exam_title": "string", "open_book": "boolean", "question_format_is_latex": "boolean", "answer_format_is_latex": "boolean" },
  "questions": [
    { "question_index": "string", "topic": "string", "keywords": ["string"], "difficulty": "string", "question_text": "string", "answer_text": "string" }
  ]
}

**元の問題構造情報:**
%s
`
