package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("codex channel: endpoint not supported")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("codex channel: /v1/messages endpoint not supported")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("codex channel: endpoint not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("codex channel: endpoint not supported")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	messages := make([]ChatMessage, 0, len(request.Messages))
	for _, msg := range request.Messages {
		cm := ChatMessage{
			Role:    msg.Role,
			Content: fmt.Sprint(msg.Content),
		}
		if msg.ToolCallId != "" {
			cm.ToolCallID = msg.ToolCallId
		}
		if msg.ToolCalls != nil {
			tcs, err := json.Marshal(msg.ToolCalls)
			if err == nil {
				json.Unmarshal(tcs, &cm.ToolCalls)
			}
		}
		if msg.Name != nil {
			cm.Name = *msg.Name
		}
		messages = append(messages, cm)
	}

	converted := ConvertMessages(messages)

	respBody := map[string]interface{}{
		"model":  request.Model,
		"input":  converted.Items,
		"stream": true,
	}

	if converted.Instructions != "" {
		respBody["instructions"] = converted.Instructions
	}
	if request.MaxTokens != nil {
		respBody["max_output_tokens"] = *request.MaxTokens
	}
	if request.Temperature != nil {
		respBody["temperature"] = *request.Temperature
	}
	if request.TopP != nil {
		respBody["top_p"] = *request.TopP
	}
	if request.Stop != nil {
		respBody["stop"] = request.Stop
	}
	if len(request.Tools) > 0 {
		toolsJSON, err := json.Marshal(request.Tools)
		if err == nil {
			respBody["tools"] = ConvertToolsRaw(toolsJSON)
		}
	}
	if request.ToolChoice != "" {
		tcJSON, err := json.Marshal(request.ToolChoice)
		if err == nil {
			respBody["tool_choice"] = ConvertToolChoiceRaw(tcJSON)
		}
	}
	respBody["store"] = false
	return json.Marshal(respBody)
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("codex channel: /v1/rerank endpoint not supported")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("codex channel: /v1/embeddings endpoint not supported")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	if len(request.Tools) > 0 {
		request.Tools = ConvertToolsRaw(request.Tools)
	}
	if len(request.ToolChoice) > 0 {
		request.ToolChoice = ConvertToolChoiceRaw(request.ToolChoice)
	}
	if info != nil && info.ChannelSetting.SystemPrompt != "" {
		systemPrompt := info.ChannelSetting.SystemPrompt
		if len(request.Instructions) == 0 {
			if b, err := common.Marshal(systemPrompt); err == nil {
				request.Instructions = b
			} else {
				return nil, err
			}
		} else if info.ChannelSetting.SystemPromptOverride {
			var existing string
			if err := common.Unmarshal(request.Instructions, &existing); err == nil {
				existing = strings.TrimSpace(existing)
				if existing == "" {
					if b, err := common.Marshal(systemPrompt); err == nil {
						request.Instructions = b
					} else {
						return nil, err
					}
				} else {
					if b, err := common.Marshal(systemPrompt + "\n" + existing); err == nil {
						request.Instructions = b
					} else {
						return nil, err
					}
				}
			}
		}
	}
	if len(request.Instructions) == 0 {
		request.Instructions = json.RawMessage(`""`)
	}
	request.Store = json.RawMessage("false")
	request.MaxOutputTokens = nil
	request.Temperature = nil
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		return openai.OaiResponsesCompactionHandler(c, resp)
	}
	if info.RelayMode == relayconstant.RelayModeResponses {
		if info.IsStream {
			return openai.OaiResponsesStreamHandler(c, info, resp)
		}
		return openai.OaiResponsesHandler(c, info, resp)
	}
	if info.RelayMode == relayconstant.RelayModeChatCompletions {
		if info.IsStream {
			if sErr := StreamTranslate(resp.Body, c.Writer, info.UpstreamModelName); sErr != nil {
				return nil, types.NewError(sErr, types.ErrorCodeInternalError)
			}
			return nil, nil
		}
		result, aErr := AggregateResponse(resp.Body)
		if aErr != nil {
			return nil, types.NewError(aErr, types.ErrorCodeInternalError)
		}
		return result, nil
	}
	return nil, types.NewError(errors.New("codex channel: endpoint not supported"), types.ErrorCodeInvalidRequest)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info.RelayMode != relayconstant.RelayModeResponses && info.RelayMode != relayconstant.RelayModeResponsesCompact && info.RelayMode != relayconstant.RelayModeChatCompletions {
		return "", errors.New("codex channel: only /v1/responses and /v1/chat/completions are supported")
	}
	path := "/backend-api/codex/responses"
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		path = "/backend-api/codex/responses/compact"
	}
	return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, path, info.ChannelType), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	key := strings.TrimSpace(info.ApiKey)
	if !strings.HasPrefix(key, "{") {
		return errors.New("codex channel: key must be a JSON object")
	}
	oauthKey, err := ParseOAuthKey(key)
	if err != nil {
		return err
	}
	accessToken := strings.TrimSpace(oauthKey.AccessToken)
	accountID := strings.TrimSpace(oauthKey.AccountID)
	if accessToken == "" {
		return errors.New("codex channel: access_token is required")
	}
	if accountID == "" {
		return errors.New("codex channel: account_id is required")
	}
	req.Set("Authorization", "Bearer "+accessToken)
	req.Set("chatgpt-account-id", accountID)
	if req.Get("OpenAI-Beta") == "" {
		req.Set("OpenAI-Beta", "responses=experimental")
	}
	if req.Get("originator") == "" {
		req.Set("originator", "codex_cli_rs")
	}
	req.Set("Content-Type", "application/json")
	if info.IsStream {
		req.Set("Accept", "text/event-stream")
	} else if req.Get("Accept") == "" {
		req.Set("Accept", "application/json")
	}
	return nil
}
