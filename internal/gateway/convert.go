package gateway

import (
	"encoding/json"
	"strings"
)

type m = map[string]any

// contentText 将 openai/anthropic 的 content（字符串或分块数组）压平为文本
func contentText(c any) string {
	switch v := c.(type) {
	case string:
		return v
	case []any:
		var sb strings.Builder
		for _, p := range v {
			if pm, ok := p.(map[string]any); ok {
				if t, ok := pm["text"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
		return sb.String()
	}
	return ""
}

func copyKeys(dst, src m, keys ...string) {
	for _, k := range keys {
		if v, ok := src[k]; ok {
			dst[k] = v
		}
	}
}

// anthropicReqToOpenAI 将 Anthropic Messages 请求转为 OpenAI Chat Completions（仅文本）
func anthropicReqToOpenAI(in m) m {
	var msgs []any
	if s := contentText(in["system"]); s != "" {
		msgs = append(msgs, m{"role": "system", "content": s})
	}
	if arr, ok := in["messages"].([]any); ok {
		for _, x := range arr {
			if xm, ok := x.(map[string]any); ok {
				msgs = append(msgs, m{"role": xm["role"], "content": contentText(xm["content"])})
			}
		}
	}
	out := m{"messages": msgs}
	copyKeys(out, in, "model", "max_tokens", "temperature", "top_p", "stream")
	if s, ok := in["stop_sequences"]; ok {
		out["stop"] = s
	}
	if st, _ := in["stream"].(bool); st {
		out["stream_options"] = m{"include_usage": true}
	}
	return out
}

// openAIReqToAnthropic 将 OpenAI 请求转为 Anthropic Messages（仅文本）
func openAIReqToAnthropic(in m) m {
	var msgs []any
	var sys []string
	if arr, ok := in["messages"].([]any); ok {
		for _, x := range arr {
			xm, ok := x.(map[string]any)
			if !ok {
				continue
			}
			role, _ := xm["role"].(string)
			text := contentText(xm["content"])
			if role == "system" || role == "developer" {
				sys = append(sys, text)
				continue
			}
			if role != "assistant" {
				role = "user"
			}
			msgs = append(msgs, m{"role": role, "content": text})
		}
	}
	out := m{"messages": msgs, "max_tokens": 4096}
	if len(sys) > 0 {
		out["system"] = strings.Join(sys, "\n")
	}
	copyKeys(out, in, "model", "temperature", "top_p", "stream")
	if v, ok := in["max_tokens"]; ok {
		out["max_tokens"] = v
	}
	if v, ok := in["max_completion_tokens"]; ok {
		out["max_tokens"] = v
	}
	switch s := in["stop"].(type) {
	case string:
		out["stop_sequences"] = []string{s}
	case []any:
		out["stop_sequences"] = s
	}
	return out
}

func anthropicStopToOpenAI(r string) string {
	switch r {
	case "max_tokens":
		return "length"
	case "":
		return "stop"
	}
	return "stop"
}

func openAIStopToAnthropic(r string) string {
	if r == "length" {
		return "max_tokens"
	}
	return "end_turn"
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

// openAIRespToAnthropic 非流式响应转换
func openAIRespToAnthropic(in m, model string) m {
	text, finish := "", ""
	if ch, ok := in["choices"].([]any); ok && len(ch) > 0 {
		if c0, ok := ch[0].(map[string]any); ok {
			if msg, ok := c0["message"].(map[string]any); ok {
				text = contentText(msg["content"])
			}
			finish, _ = c0["finish_reason"].(string)
		}
	}
	u, _ := in["usage"].(map[string]any)
	id, _ := in["id"].(string)
	return m{
		"id": id, "type": "message", "role": "assistant", "model": model,
		"content":     []any{m{"type": "text", "text": text}},
		"stop_reason": openAIStopToAnthropic(finish),
		"usage":       m{"input_tokens": num(u["prompt_tokens"]), "output_tokens": num(u["completion_tokens"])},
	}
}

func anthropicRespToOpenAI(in m, model string) m {
	text := contentText(in["content"])
	u, _ := in["usage"].(map[string]any)
	in_, out_ := num(u["input_tokens"]), num(u["output_tokens"])
	id, _ := in["id"].(string)
	sr, _ := in["stop_reason"].(string)
	return m{
		"id": id, "object": "chat.completion", "model": model,
		"choices": []any{m{"index": 0, "message": m{"role": "assistant", "content": text}, "finish_reason": anthropicStopToOpenAI(sr)}},
		"usage":   m{"prompt_tokens": in_, "completion_tokens": out_, "total_tokens": in_ + out_},
	}
}

func jsonBytes(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
