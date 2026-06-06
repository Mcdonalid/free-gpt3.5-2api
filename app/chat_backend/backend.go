package chat_backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"chat2api/app/acc_token_pool"
	"chat2api/app/common"
	"chat2api/app/conf"
	"chat2api/app/constant"
	"chat2api/app/proof_work"
	"chat2api/app/turnstile"

	"github.com/aurorax-neo/tls_client_httpi"
	"github.com/aurorax-neo/tls_client_httpi/tls_client"
	"github.com/google/uuid"
)

type Backend struct {
	HTTP      tls_client_httpi.TCHI
	Auth      *chatRequirements
	AccAuth   string
	BaseURL   string
	ChatURL   string
	UserAgent string
	SessionID string
	Cookies   tls_client_httpi.Cookies
	Pow       proof_work.Resources
}

type chatRequirements struct {
	OaiDeviceID    string               `json:"-"`
	Arkose         challenge            `json:"arkose"`
	Turnstile      challenge            `json:"turnstile"`
	TurnstileToken string               `json:"-"`
	ProofWork      proof_work.ProofWork `json:"proofofwork"`
	Token          string               `json:"token"`
	SoToken        string               `json:"so_token"`
	ForceLogin     bool                 `json:"force_login"`
}

type challenge struct {
	Required bool   `json:"required"`
	Dx       string `json:"dx"`
}

func New(token string, retry int) (*Backend, error) {
	token = strings.TrimSpace(token)
	localToken := strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	if strings.HasPrefix(localToken, "at-") {
		return newBackend("Bearer "+strings.TrimPrefix(localToken, "at-"), "")
	}
	if strings.HasPrefix(token, "Bearer eyJhbGciOiJSUzI1NiI") {
		return newBackend(token, "")
	}
	if !acc_token_pool.GetAccAuthPoolInstance().IsEmpty() {
		accessToken := acc_token_pool.GetAccAuthPoolInstance().GetAccessToken()
		if accessToken == nil || accessToken.Token == "" {
			return nil, fmt.Errorf("access token pool is empty")
		}
		backend, err := newBackend(accessToken.Token, accessToken.Proxy)
		if backend == nil && retry > 0 {
			return New(token, retry-1)
		}
		return backend, err
	}
	if strings.HasPrefix(localToken, "sk-") {
		return nil, fmt.Errorf("access token pool is empty")
	}
	backend, err := newBackend(token, "")
	if backend == nil && retry > 0 {
		return New(token, retry-1)
	}
	return backend, err
}

func newBackend(token string, accountProxy string) (*Backend, error) {
	appConf := conf.GetApp()
	baseURL := strings.TrimRight(appConf.ChatGPTBaseUrl, "/")
	if baseURL == "" {
		baseURL = "https://chatgpt.com"
	}
	b := &Backend{
		HTTP:      tls_client.NewClient(tls_client.NewClientOptions(300, common.GetClientProfile())),
		Auth:      &chatRequirements{OaiDeviceID: uuid.New().String()},
		BaseURL:   baseURL,
		ChatURL:   baseURL + "/backend-anon/conversation",
		UserAgent: common.GetUa(),
		SessionID: uuid.New().String(),
	}
	if b.HTTP == nil {
		return nil, fmt.Errorf("http client is nil")
	}
	if strings.HasPrefix(token, "Bearer ") {
		b.AccAuth = token
		b.ChatURL = baseURL + "/backend-api/conversation"
	}
	proxy := strings.TrimSpace(accountProxy)
	if proxy == "" {
		proxy = strings.TrimSpace(appConf.Proxy)
	}
	if proxy != "" {
		if err := b.HTTP.SetProxy(proxy); err != nil {
			return nil, err
		}
	}
	b.loadPowResources()
	if err := b.loadRequirements(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Backend) Headers(url string) (tls_client_httpi.Headers, tls_client_httpi.Cookies) {
	headers := tls_client_httpi.Headers{}
	path := strings.TrimPrefix(url, b.BaseURL)
	headers.Set("accept", "*/*")
	headers.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7")
	headers.Set("origin", b.BaseURL)
	headers.Set("referer", b.BaseURL+"/")
	headers.Set("cache-control", "no-cache")
	headers.Set("pragma", "no-cache")
	headers.Set("priority", "u=1, i")
	headers.Set("sec-ch-ua", `"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"`)
	headers.Set("sec-ch-ua-arch", `"x86"`)
	headers.Set("sec-ch-ua-bitness", `"64"`)
	headers.Set("sec-ch-ua-full-version", `"143.0.3650.96"`)
	headers.Set("sec-ch-ua-full-version-list", `"Microsoft Edge";v="143.0.3650.96", "Chromium";v="143.0.7499.147", "Not A(Brand";v="24.0.0.0"`)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-model", `""`)
	headers.Set("sec-ch-ua-platform", `"Windows"`)
	headers.Set("sec-ch-ua-platform-version", `"19.0.0"`)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	headers.Set("user-agent", b.UserAgent)
	headers.Set("oai-device-id", b.Auth.OaiDeviceID)
	headers.Set("oai-session-id", b.SessionID)
	headers.Set("oai-language", "zh-CN")
	headers.Set("oai-client-version", "prod-3b8f2c1740596d77c64c1d3d50205828839b2730")
	headers.Set("oai-client-build-number", "3310101057")
	headers.Set("x-openai-target-path", path)
	headers.Set("x-openai-target-route", path)
	if b.AccAuth != "" {
		headers.Set("authorization", b.AccAuth)
	}
	return headers, b.Cookies
}

func (b *Backend) loadPowResources() {
	headers, cookies := b.Headers(b.BaseURL + "/")
	headers.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	response, err := b.HTTP.Request(tls_client_httpi.GET, b.BaseURL+"/", headers, cookies, nil)
	if err != nil {
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return
	}
	b.Pow = proof_work.ParseResources(string(body))
}

func (b *Backend) loadRequirements() error {
	authURL := b.BaseURL + "/backend-anon/sentinel/chat-requirements"
	if b.AccAuth != "" {
		authURL = b.BaseURL + "/backend-api/sentinel/chat-requirements"
	}
	requirementsToken := proof_work.LegacyRequirementsToken(b.UserAgent, b.Pow)
	body := bytes.NewBufferString(`{"p":"` + requirementsToken + `"}`)
	headers, cookies := b.Headers(authURL)
	headers.Set("content-type", "application/json")
	response, err := b.HTTP.Request(tls_client_httpi.POST, authURL, headers, cookies, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("chat requirements failed: status=%d body=%s", response.StatusCode, string(detail))
	}
	if err := json.NewDecoder(response.Body).Decode(&b.Auth); err != nil {
		return err
	}
	if b.Auth.ForceLogin {
		common.SubUpdateThreshold()
		return fmt.Errorf("force login required")
	}
	if b.Auth.Arkose.Required {
		return fmt.Errorf("arkose token is required")
	}
	if b.Auth.Turnstile.Required && b.Auth.Turnstile.Dx != "" {
		sourceP := ""
		if b.AccAuth == "" {
			sourceP = requirementsToken
		}
		b.Auth.TurnstileToken = turnstile.Solve(b.Auth.Turnstile.Dx, sourceP)
		if b.Auth.TurnstileToken == "" {
			fallbackP := requirementsToken
			if sourceP == requirementsToken {
				fallbackP = ""
			}
			b.Auth.TurnstileToken = turnstile.Solve(b.Auth.Turnstile.Dx, fallbackP)
		}
	}
	if b.Auth.ProofWork.Required {
		b.Auth.ProofWork.Ospt = proof_work.CalcProofToken(b.Auth.ProofWork.Seed, b.Auth.ProofWork.Difficulty, b.UserAgent, b.Pow)
		if b.Auth.ProofWork.Ospt == "" {
			return fmt.Errorf("proof token failed")
		}
	}
	if b.Auth.Token == "" {
		return fmt.Errorf("missing chat requirements token")
	}
	return nil
}

func Retry() int {
	return constant.ReTry
}
