package gateway

import (
	"bufio"
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Server struct {
	Pool        *Pool
	ClientKey   string // 调用方访问中转服务所需的 key，空则不校验
	AdminToken  string // 管理页面密码（Basic Auth 密码 / Bearer）
	HTTPClient  *http.Client
	adminAssets []byte
}

func NewServer(p *Pool, clientKey, adminToken string) *Server {
	return &Server{Pool: p, ClientKey: clientKey, AdminToken: adminToken, HTTPClient: &http.Client{}}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", s.clientAuth(func(w http.ResponseWriter, r *http.Request) { s.proxy(w, r, "openai") }))
	mux.HandleFunc("POST /v1/messages", s.clientAuth(func(w http.ResponseWriter, r *http.Request) { s.proxy(w, r, "anthropic") }))
	mux.HandleFunc("GET /v1/models", s.clientAuth(s.models))
	mux.HandleFunc("GET /admin", s.adminAuth(s.adminPage))
	mux.HandleFunc("GET /admin/api/keys", s.adminAuth(s.listKeys))
	mux.HandleFunc("POST /admin/api/keys", s.adminAuth(s.addKey))
	mux.HandleFunc("DELETE /admin/api/keys/{id}", s.adminAuth(s.deleteKey))
	mux.HandleFunc("POST /admin/api/keys/{id}/enabled", s.adminAuth(s.toggleKey))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/admin", http.StatusFound) })
	return mux
}

func eq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

func (s *Server) clientAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.ClientKey != "" {
			k := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if k == "" {
				k = r.Header.Get("x-api-key")
			}
			if !eq(k, s.ClientKey) {
				writeErr(w, http.StatusUnauthorized, "invalid gateway api key")
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.AdminToken == "" {
			http.Error(w, "ADMIN_TOKEN is not configured; admin disabled", http.StatusForbidden)
			return
		}
		_, pw, ok := r.BasicAuth()
		if !ok {
			pw = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if !eq(pw, s.AdminToken) {
			w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(jsonBytes(m{"error": m{"message": msg, "type": "gateway_error"}}))
}

func (s *Server) models(w http.ResponseWriter, r *http.Request) {
	seen := map[string]bool{}
	data := []any{}
	for _, e := range s.Pool.List() {
		if e.Enabled && e.Model != "" && !seen[e.Model] {
			seen[e.Model] = true
			data = append(data, m{"id": e.Model, "object": "model", "owned_by": "gateway"})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonBytes(m{"object": "list", "data": data}))
}

func (s *Server) proxy(w http.ResponseWriter, r *http.Request, clientProto string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		writeErr(w, 400, "cannot read body")
		return
	}
	var req m
	if err := json.Unmarshal(body, &req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	stream, _ := req["stream"].(bool)
	reqModel, _ := req["model"].(string)

	cands := s.Pool.Candidates()
	if len(cands) == 0 {
		writeErr(w, http.StatusServiceUnavailable, "no available api key in pool")
		return
	}
	lastStatus, lastMsg := http.StatusBadGateway, "all upstream keys failed"
	for _, e := range cands {
		var out m
		switch {
		case e.Protocol == clientProto:
			out = cloneMap(req)
		case clientProto == "anthropic":
			out = anthropicReqToOpenAI(req)
		default:
			out = openAIReqToAnthropic(req)
		}
		if e.Model != "" {
			out["model"] = e.Model
		}
		upModel, _ := out["model"].(string)

		url, hdr := e.BaseURL+"/chat/completions", http.Header{}
		if e.Protocol == "anthropic" {
			url = e.BaseURL + "/messages"
			hdr.Set("x-api-key", e.APIKey)
			v := r.Header.Get("anthropic-version")
			if v == "" {
				v = "2023-06-01"
			}
			hdr.Set("anthropic-version", v)
		} else {
			hdr.Set("Authorization", "Bearer "+e.APIKey)
		}
		hdr.Set("Content-Type", "application/json")
		up, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(jsonBytes(out)))
		up.Header = hdr
		resp, err := s.HTTPClient.Do(up)
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			s.Pool.Report(e.ID, false, false, err.Error())
			lastMsg = err.Error()
			continue
		}
		if resp.StatusCode >= 400 {
			eb, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			msg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(eb), 300))
			if resp.StatusCode == 400 || resp.StatusCode == 422 {
				// 请求本身有问题，原样返回，不惩罚 key
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(eb)
				return
			}
			fatal := resp.StatusCode == 401 || resp.StatusCode == 402 || resp.StatusCode == 403
			s.Pool.Report(e.ID, false, fatal, msg)
			lastStatus, lastMsg = resp.StatusCode, msg
			continue
		}
		s.Pool.Report(e.ID, true, false, "")
		if reqModel == "" {
			reqModel = upModel
		}
		defer resp.Body.Close()
		if stream {
			s.relayStream(w, resp.Body, e.Protocol, clientProto, reqModel)
			return
		}
		rb, err := io.ReadAll(resp.Body)
		if err != nil {
			writeErr(w, 502, err.Error())
			return
		}
		if e.Protocol != clientProto {
			var in m
			if json.Unmarshal(rb, &in) == nil {
				if clientProto == "anthropic" {
					rb = jsonBytes(openAIRespToAnthropic(in, reqModel))
				} else {
					rb = jsonBytes(anthropicRespToOpenAI(in, reqModel))
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(rb)
		return
	}
	if lastStatus < 400 || lastStatus == 401 || lastStatus == 402 || lastStatus == 403 {
		lastStatus = http.StatusBadGateway
	}
	writeErr(w, lastStatus, lastMsg)
}

func cloneMap(in m) m {
	out := make(m, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---- 流式 ----

type sseWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func (s sseWriter) event(name string, data any) {
	if name != "" {
		fmt.Fprintf(s.w, "event: %s\n", name)
	}
	fmt.Fprintf(s.w, "data: %s\n\n", jsonBytes(data))
	if s.f != nil {
		s.f.Flush()
	}
}

func (s sseWriter) raw(line string) {
	fmt.Fprint(s.w, line)
	if s.f != nil {
		s.f.Flush()
	}
}

func (s *Server) relayStream(w http.ResponseWriter, body io.Reader, up, client, model string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fl, _ := w.(http.Flusher)
	sw := sseWriter{w, fl}
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 64*1024), 4<<20)

	if up == client {
		for sc.Scan() {
			sw.raw(sc.Text() + "\n")
		}
		return
	}

	if up == "openai" { // openai 流 -> anthropic 流
		sw.event("message_start", m{"type": "message_start", "message": m{"id": "msg_gw", "type": "message", "role": "assistant", "model": model, "content": []any{}, "usage": m{"input_tokens": 0, "output_tokens": 0}}})
		sw.event("content_block_start", m{"type": "content_block_start", "index": 0, "content_block": m{"type": "text", "text": ""}})
		finish, outTok := "", 0
		for sc.Scan() {
			d, ok := strings.CutPrefix(sc.Text(), "data:")
			d = strings.TrimSpace(d)
			if !ok || d == "" || d == "[DONE]" {
				continue
			}
			var ch m
			if json.Unmarshal([]byte(d), &ch) != nil {
				continue
			}
			if u, ok := ch["usage"].(map[string]any); ok {
				outTok = num(u["completion_tokens"])
			}
			if cs, ok := ch["choices"].([]any); ok && len(cs) > 0 {
				c0, _ := cs[0].(map[string]any)
				if dl, ok := c0["delta"].(map[string]any); ok {
					if t, _ := dl["content"].(string); t != "" {
						sw.event("content_block_delta", m{"type": "content_block_delta", "index": 0, "delta": m{"type": "text_delta", "text": t}})
					}
				}
				if f, _ := c0["finish_reason"].(string); f != "" {
					finish = f
				}
			}
		}
		sw.event("content_block_stop", m{"type": "content_block_stop", "index": 0})
		sw.event("message_delta", m{"type": "message_delta", "delta": m{"stop_reason": openAIStopToAnthropic(finish)}, "usage": m{"output_tokens": outTok}})
		sw.event("message_stop", m{"type": "message_stop"})
		return
	}

	// anthropic 流 -> openai 流
	id := "chatcmpl-gw"
	chunk := func(delta m, finish any) m {
		return m{"id": id, "object": "chat.completion.chunk", "model": model, "choices": []any{m{"index": 0, "delta": delta, "finish_reason": finish}}}
	}
	sw.event("", chunk(m{"role": "assistant", "content": ""}, nil))
	for sc.Scan() {
		d, ok := strings.CutPrefix(sc.Text(), "data:")
		d = strings.TrimSpace(d)
		if !ok || d == "" {
			continue
		}
		var ev m
		if json.Unmarshal([]byte(d), &ev) != nil {
			continue
		}
		switch ev["type"] {
		case "message_start":
			if mm, ok := ev["message"].(map[string]any); ok {
				if s, _ := mm["id"].(string); s != "" {
					id = s
				}
			}
		case "content_block_delta":
			if dl, ok := ev["delta"].(map[string]any); ok {
				if t, _ := dl["text"].(string); t != "" {
					sw.event("", chunk(m{"content": t}, nil))
				}
			}
		case "message_delta":
			if dl, ok := ev["delta"].(map[string]any); ok {
				sr, _ := dl["stop_reason"].(string)
				sw.event("", chunk(m{}, anthropicStopToOpenAI(sr)))
			}
		}
	}
	sw.raw("data: [DONE]\n\n")
}

// ---- 管理接口 ----

func maskKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	list := s.Pool.List()
	for i := range list {
		list[i].APIKey = maskKey(list[i].APIKey)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonBytes(list))
}

func (s *Server) addKey(w http.ResponseWriter, r *http.Request) {
	var e Entry
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&e); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	added, err := s.Pool.Add(e)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	added.APIKey = maskKey(added.APIKey)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonBytes(added))
}

func (s *Server) deleteKey(w http.ResponseWriter, r *http.Request) {
	if !s.Pool.Delete(r.PathValue("id")) {
		writeErr(w, 404, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) toggleKey(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Enabled bool `json:"enabled"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1<<10)).Decode(&b) != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if !s.Pool.SetEnabled(r.PathValue("id"), b.Enabled) {
		writeErr(w, 404, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}
