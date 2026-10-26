package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

// Entry 是池中的一个上游 API Key
type Entry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	APIKey      string `json:"api_key"`
	BaseURL     string `json:"base_url"`
	Protocol    string `json:"protocol"`        // 上游协议: openai | anthropic
	Model       string `json:"model,omitempty"` // 非空时覆盖请求里的模型名
	Enabled     bool   `json:"enabled"`
	ErrorStreak int    `json:"error_streak"`
	LastError   string `json:"last_error,omitempty"`
}

const maxStreak = 3

// Pool 线程安全、可在运行时增删 Key 的轮询池，并持久化到 JSON 文件
type Pool struct {
	mu      sync.Mutex
	entries []*Entry
	next    int
	path    string
}

func NewPool(path string) (*Pool, error) {
	p := &Pool{path: path}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return p, nil
		}
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &p.entries); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *Pool) save() error {
	b, err := json.MarshalIndent(p.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path, b, 0o600)
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (p *Pool) Add(e Entry) (*Entry, error) {
	e.Protocol = strings.ToLower(strings.TrimSpace(e.Protocol))
	if e.Protocol == "" {
		e.Protocol = "openai"
	}
	if e.Protocol != "openai" && e.Protocol != "anthropic" {
		return nil, errors.New("protocol must be openai or anthropic")
	}
	e.APIKey = strings.TrimSpace(e.APIKey)
	e.BaseURL = strings.TrimRight(strings.TrimSpace(e.BaseURL), "/")
	if e.APIKey == "" || e.BaseURL == "" {
		return nil, errors.New("api_key and base_url are required")
	}
	if !strings.HasPrefix(e.BaseURL, "http://") && !strings.HasPrefix(e.BaseURL, "https://") {
		return nil, errors.New("base_url must start with http:// or https://")
	}
	e.ID = newID()
	e.Enabled = true
	e.ErrorStreak = 0
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = append(p.entries, &e)
	if err := p.save(); err != nil {
		return nil, err
	}
	c := e
	return &c, nil
}

func (p *Pool) Delete(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, e := range p.entries {
		if e.ID == id {
			p.entries = append(p.entries[:i], p.entries[i+1:]...)
			_ = p.save()
			return true
		}
	}
	return false
}

func (p *Pool) SetEnabled(id string, enabled bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		if e.ID == id {
			e.Enabled = enabled
			if enabled {
				e.ErrorStreak = 0
				e.LastError = ""
			}
			_ = p.save()
			return true
		}
	}
	return false
}

// List 返回快照
func (p *Pool) List() []Entry {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Entry, 0, len(p.entries))
	for _, e := range p.entries {
		out = append(out, *e)
	}
	return out
}

// Candidates 以轮询顺序返回所有启用的 Key（每次调用起点前进一位）
func (p *Pool) Candidates() []Entry {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.entries)
	var out []Entry
	for i := 0; i < n; i++ {
		e := p.entries[(p.next+i)%n]
		if e.Enabled {
			out = append(out, *e)
		}
	}
	if n > 0 {
		p.next = (p.next + 1) % n
	}
	return out
}

// Report 汇报一次调用结果。fatal 表示认证/额度类错误，直接禁用。
func (p *Pool) Report(id string, ok, fatal bool, msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		if e.ID != id {
			continue
		}
		if ok {
			e.ErrorStreak = 0
			e.LastError = ""
		} else {
			e.ErrorStreak++
			e.LastError = msg
			if fatal || e.ErrorStreak >= maxStreak {
				e.Enabled = false
			}
		}
		_ = p.save()
		return
	}
}
