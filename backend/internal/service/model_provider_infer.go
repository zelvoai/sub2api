package service

import (
	"net/url"
	"strings"
)

func InferModelProvider(modelName, fallbackHost string) string {
	lowered := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.HasPrefix(lowered, "gpt"), strings.HasPrefix(lowered, "o1"), strings.HasPrefix(lowered, "o3"), strings.HasPrefix(lowered, "o4"), strings.HasPrefix(lowered, "chatgpt"):
		return "OpenAI"
	case strings.HasPrefix(lowered, "claude"):
		return "Anthropic"
	case strings.HasPrefix(lowered, "gemini"):
		return "Gemini"
	case strings.HasPrefix(lowered, "deepseek"):
		return "DeepSeek"
	case strings.HasPrefix(lowered, "qwen"), strings.HasPrefix(lowered, "qwq"):
		return "Qwen"
	case strings.HasPrefix(lowered, "glm"):
		return "Zhipu"
	case strings.HasPrefix(lowered, "moonshot"), strings.HasPrefix(lowered, "kimi"):
		return "Moonshot"
	case strings.HasPrefix(lowered, "grok"):
		return "xAI"
	}
	if host := upstreamHostName(fallbackHost); host != "" {
		return "Upstream: " + host
	}
	return "Custom"
}

func upstreamHostName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Hostname() == "" {
		return strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
	}
	return u.Hostname()
}
