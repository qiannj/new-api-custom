package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// ChatFormatTool represents a tool in Chat Completions format (with function wrapper)
type ChatFormatTool struct {
	Type     string              `json:"type"`
	Function *ChatFormatFunction `json:"function,omitempty"`
}

type ChatFormatFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

// ResponsesFormatTool represents a tool in Responses API format (flattened)
type ResponsesFormatTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

// ConvertToolsRaw converts tools from Chat format (with function wrapper) to Responses format (flattened).
func ConvertToolsRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var chatTools []ChatFormatTool
	if err := common.Unmarshal(raw, &chatTools); err != nil {
		return raw
	}
	out := make([]ResponsesFormatTool, 0, len(chatTools))
	for _, t := range chatTools {
		if t.Function != nil {
			out = append(out, ResponsesFormatTool{
				Type:        "function",
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
				Strict:      t.Function.Strict,
			})
		} else {
			out = append(out, ResponsesFormatTool{
				Type: t.Type,
			})
		}
	}
	result, _ := common.Marshal(out)
	return result
}

// ConvertToolChoiceRaw converts tool_choice from Chat format to Responses format.
func ConvertToolChoiceRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var str string
	if err := common.Unmarshal(raw, &str); err == nil {
		return raw
	}
	var tc map[string]interface{}
	if err := common.Unmarshal(raw, &tc); err != nil {
		return raw
	}
	if tc == nil {
		return raw
	}
	tcType, _ := tc["type"].(string)
	if tcType != "function" {
		return raw
	}
	fn, ok := tc["function"].(map[string]interface{})
	if !ok {
		return raw
	}
	name, _ := fn["name"].(string)
	if name == "" {
		return raw
	}
	result, _ := common.Marshal(map[string]interface{}{
		"type": tcType,
		"name": name,
	})
	return result
}

// ChatMessage represents a single message in Chat Completions format.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ConvertedMessages holds the converted Responses API input items and instructions.
type ConvertedMessages struct {
	Items        []MessageItem `json:"items"`
	Instructions string        `json:"instructions,omitempty"`
}

type MessageItem struct {
	Type      string `json:"type"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Status    string `json:"status,omitempty"`
	Output    string `json:"output,omitempty"`
}

// ConvertMessages converts messages from Chat Completions format to Responses API format.
func ConvertMessages(messages []ChatMessage) ConvertedMessages {
	var sysMsgs []string
	var items []MessageItem
	for _, m := range messages {
		if m.Role == "system" {
			sysMsgs = append(sysMsgs, m.Content)
			continue
		}
		if m.Role == "tool" {
			items = append(items, MessageItem{
				Type:   "function_call_output",
				CallID: m.ToolCallID,
				Output: m.Content,
			})
			continue
		}
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			if m.Content != "" {
				items = append(items, MessageItem{
					Type:    "message",
					Role:    "assistant",
					Content: m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				items = append(items, MessageItem{
					Type:      "function_call",
					CallID:    tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
					Status:    "completed",
				})
			}
			continue
		}
		items = append(items, MessageItem{
			Type:    "message",
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return ConvertedMessages{
		Items:        items,
		Instructions: strings.Join(sysMsgs, "\n"),
	}
}

// SSEEvent represents a parsed Responses API SSE event.
type SSEEvent struct {
	Type        string              `json:"type"`
	Delta       string              `json:"delta,omitempty"`
	ItemID      string              `json:"item_id,omitempty"`
	OutputIndex *int                `json:"output_index,omitempty"`
	Item        *ResponsesItem      `json:"item,omitempty"`
	Response    *ResponsesCompleted `json:"response,omitempty"`
	Arguments   string              `json:"arguments,omitempty"`
}

type ResponsesItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Role      string `json:"role,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ResponsesCompleted struct {
	ID     string          `json:"id"`
	Model  string          `json:"model"`
	Status string          `json:"status"`
	Usage  *ResponsesUsage `json:"usage,omitempty"`
}

type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// StreamTranslate translates Responses API SSE events to Chat Completions SSE format.
func StreamTranslate(upstream io.ReadCloser, w io.Writer, model string) error {
	if upstream == nil {
		return fmt.Errorf("upstream reader is nil")
	}

	type toolSlot struct {
		callID    string
		name      string
		tcIndex   int
		arguments string
	}

	chatID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
	currentModel := model
	toolSlots := make(map[int]*toolSlot)
	nextTcIndex := 0
	sentRole := false

	emit := func(deltaObj interface{}) ([]byte, error) {
		chunk := map[string]interface{}{
			"id":      chatID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   currentModel,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         deltaObj,
					"finish_reason": nil,
				},
			},
		}
		data, err := json.Marshal(chunk)
		if err != nil {
			return nil, err
		}
		return append([]byte("data: "), append(data, []byte("\n\n")...)...), nil
	}

	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(line[6:])
		if data == "" || data == "[DONE]" {
			continue
		}

		var evt SSEEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "response.output_item.added":
			if evt.Item != nil && evt.Item.Type == "function_call" {
				idx := nextTcIndex
				nextTcIndex++
				oi := 0
				if evt.OutputIndex != nil {
					oi = *evt.OutputIndex
				}
				toolSlots[oi] = &toolSlot{
					callID:  evt.Item.CallID,
					name:    evt.Item.Name,
					tcIndex: idx,
				}
				if !sentRole {
					sentRole = true
					chunk, err := emit(map[string]interface{}{"role": "assistant", "content": ""})
					if err != nil {
						return err
					}
					if _, err := w.Write(chunk); err != nil {
						return err
					}
				}
				chunk, err := emit(map[string]interface{}{
					"tool_calls": []map[string]interface{}{
						{
							"index":    idx,
							"id":       evt.Item.CallID,
							"type":     "function",
							"function": map[string]string{"name": evt.Item.Name, "arguments": ""},
						},
					},
				})
				if err != nil {
					return err
				}
				if _, err := w.Write(chunk); err != nil {
					return err
				}
			}

		case "response.output_text.delta":
			if !sentRole {
				sentRole = true
				chunk, err := emit(map[string]interface{}{"role": "assistant", "content": ""})
				if err != nil {
					return err
				}
				if _, err := w.Write(chunk); err != nil {
					return err
				}
			}
			chunk, err := emit(map[string]interface{}{"content": evt.Delta})
			if err != nil {
				return err
			}
			if _, err := w.Write(chunk); err != nil {
				return err
			}

		case "response.function_call_arguments.delta":
			oi := 0
			if evt.OutputIndex != nil {
				oi = *evt.OutputIndex
			}
			slot, ok := toolSlots[oi]
			if !ok {
				continue
			}
			slot.arguments += evt.Delta
			chunk, err := emit(map[string]interface{}{
				"tool_calls": []map[string]interface{}{
					{
						"index":    slot.tcIndex,
						"function": map[string]string{"arguments": evt.Delta},
					},
				},
			})
			if err != nil {
				return err
			}
			if _, err := w.Write(chunk); err != nil {
				return err
			}

		case "response.function_call_arguments.done":

		case "response.output_item.done":

		case "response.completed":
			if evt.Response != nil {
				if evt.Response.Model != "" {
					currentModel = evt.Response.Model
				}
				finishReason := "stop"
				if evt.Response.Status == "incomplete" {
					finishReason = "length"
				} else if evt.Response.Status == "failed" {
					finishReason = "error"
				} else if len(toolSlots) > 0 {
					finishReason = "tool_calls"
				}
				finalChunk := map[string]interface{}{
					"id":      chatID,
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   currentModel,
					"choices": []map[string]interface{}{
						{
							"index":         0,
							"delta":         map[string]interface{}{},
							"finish_reason": finishReason,
						},
					},
				}
				if evt.Response.Usage != nil {
					finalChunk["usage"] = map[string]int{
						"prompt_tokens":     evt.Response.Usage.InputTokens,
						"completion_tokens": evt.Response.Usage.OutputTokens,
						"total_tokens":      evt.Response.Usage.InputTokens + evt.Response.Usage.OutputTokens,
					}
				}
				data, err := json.Marshal(finalChunk)
				if err != nil {
					return err
				}
				if _, err := w.Write(append([]byte("data: "), append(data, []byte("\n\n")...)...)); err != nil {
					return err
				}
			}

		default:
		}
	}

	if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
		return err
	}
	return scanner.Err()
}

// AggregateResponse accumulates Responses API SSE events into a single Chat Completions response.
func AggregateResponse(upstream io.ReadCloser) (*dto.OpenAITextResponse, error) {
	if upstream == nil {
		return nil, fmt.Errorf("upstream reader is nil")
	}

	type toolCallInfo struct {
		id        string
		name      string
		arguments string
	}

	var textParts []string
	toolCalls := make(map[int]*toolCallInfo)
	var finishReason string
	var usage *ResponsesUsage
	var model string
	var respID string

	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(line[6:])
		if data == "" || data == "[DONE]" {
			continue
		}

		var evt SSEEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		if evt.Response != nil && evt.Response.Model != "" {
			model = evt.Response.Model
		}

		switch evt.Type {
		case "response.output_text.delta":
			textParts = append(textParts, evt.Delta)

		case "response.function_call_arguments.delta":
			oi := 0
			if evt.OutputIndex != nil {
				oi = *evt.OutputIndex
			}
			if _, ok := toolCalls[oi]; !ok {
				toolCalls[oi] = &toolCallInfo{}
			}
			toolCalls[oi].arguments += evt.Delta

		case "response.output_item.added":
			if evt.Item != nil && evt.Item.Type == "function_call" {
				oi := 0
				if evt.OutputIndex != nil {
					oi = *evt.OutputIndex
				}
				toolCalls[oi] = &toolCallInfo{
					id:   evt.Item.CallID,
					name: evt.Item.Name,
				}
			}

		case "response.function_call_arguments.done":
			oi := 0
			if evt.OutputIndex != nil {
				oi = *evt.OutputIndex
			}
			if _, ok := toolCalls[oi]; !ok {
				toolCalls[oi] = &toolCallInfo{}
			}
			if evt.Arguments != "" {
				toolCalls[oi].arguments = evt.Arguments
			}

		case "response.output_item.done":
			if evt.Item != nil && evt.Item.Type == "function_call" {
				oi := 0
				if evt.OutputIndex != nil {
					oi = *evt.OutputIndex
				}
				callID := evt.Item.CallID
				if callID == "" {
					callID = evt.Item.ID
				}
				args := evt.Item.Arguments
				if existing, ok := toolCalls[oi]; ok && existing.arguments != "" {
					args = existing.arguments
				}
				toolCalls[oi] = &toolCallInfo{
					id:        callID,
					name:      evt.Item.Name,
					arguments: args,
				}
			}

		case "response.completed":
			if evt.Response != nil {
				model = evt.Response.Model
				if model == "" && evt.Response.Model != "" {
					model = evt.Response.Model
				}
				respID = evt.Response.ID
				if len(toolCalls) > 0 {
					finishReason = "tool_calls"
				} else if evt.Response.Status == "completed" {
					finishReason = "stop"
				} else {
					finishReason = "length"
				}
				usage = evt.Response.Usage
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if finishReason == "" {
		finishReason = "stop"
	}
	if respID == "" {
		respID = fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
	}
	if model == "" {
		model = "unknown"
	}

	message := dto.Message{
		Role: "assistant",
	}
	text := strings.Join(textParts, "")
	if text != "" {
		message.SetStringContent(text)
	} else {
		message.SetNullContent()
	}

	if len(toolCalls) > 0 {
		chatToolCalls := make([]dto.ToolCallRequest, 0, len(toolCalls))
		for idx, tc := range toolCalls {
			callID := tc.id
			if callID == "" {
				callID = fmt.Sprintf("call_%d", idx)
			}
			chatToolCalls = append(chatToolCalls, dto.ToolCallRequest{
				ID:   callID,
				Type: "function",
				Function: dto.FunctionRequest{
					Name:      tc.name,
					Arguments: tc.arguments,
				},
			})
		}
		message.SetToolCalls(chatToolCalls)
	}

	usageOut := dto.Usage{}
	if usage != nil {
		usageOut.PromptTokens = usage.InputTokens
		usageOut.CompletionTokens = usage.OutputTokens
		usageOut.TotalTokens = usage.InputTokens + usage.OutputTokens
	}

	return &dto.OpenAITextResponse{
		Id:      respID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index:        0,
				Message:      message,
				FinishReason: finishReason,
			},
		},
		Usage: usageOut,
	}, nil
}
