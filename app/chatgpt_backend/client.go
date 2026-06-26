package chatgpt_backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"chat2api/app/common"
	"chat2api/app/conf"
	"chat2api/app/constant"
	"chat2api/app/token_pool"

	"github.com/aurorax-neo/tls_client_httpi"
	"github.com/aurorax-neo/tls_client_httpi/tls_client"
	"github.com/google/uuid"
)

type Client struct {
	HTTP      tls_client_httpi.TCHI
	Auth      *chatRequirements
	AccAuth   string
	BaseURL   string
	ChatURL   string
	UserAgent string
	SessionID string
	DeviceID  string
	Cookies   tls_client_httpi.Cookies
	Pow       Resources
}

type chatRequirements struct {
	Persona        string    `json:"persona,omitempty"`
	Token          string    `json:"token"`
	PrepareToken   string    `json:"prepare_token,omitempty"`
	Arkose         challenge `json:"arkose"`
	Turnstile      challenge `json:"turnstile"`
	TurnstileToken string    `json:"-"`
	ProofWork      ProofWork `json:"proofofwork"`
	ForceLogin     bool      `json:"force_login"`
}

type challenge struct {
	Required bool   `json:"required"`
	Dx       string `json:"dx"`
}

func New(token string, retry int) (*Client, error) {
	token = strings.TrimSpace(token)
	localToken := strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	appConf := conf.GetApp()
	if accessToken, ok := appConf.DirectAccessToken(localToken); ok {
		return newClient("Bearer "+accessToken, "")
	}
	if strings.HasPrefix(token, "Bearer eyJhbGciOiJSUzI1NiI") {
		return newClient(token, "")
	}
	if !token_pool.GetAccessTokenPool().IsEmpty() {
		accessToken := token_pool.GetAccessTokenPool().GetAccessToken()
		if accessToken == nil || accessToken.Token == "" {
			return nil, fmt.Errorf("access token pool is empty")
		}
		client, err := newClient(accessToken.Token, accessToken.Proxy)
		if client == nil && retry > 0 {
			return New(token, retry-1)
		}
		return client, err
	}
	if strings.HasPrefix(localToken, "sk-") {
		return nil, fmt.Errorf("access token pool is empty")
	}
	client, err := newClient(token, "")
	if client == nil && retry > 0 {
		return New(token, retry-1)
	}
	return client, err
}

func newClient(token string, accountProxy string) (*Client, error) {
	appConf := conf.GetApp()
	baseURL := strings.TrimRight(appConf.ChatGPTBaseUrl, "/")
	if baseURL == "" {
		baseURL = "https://chatgpt.com"
	}
	deviceID := uuid.New().String()
	c := &Client{
		HTTP:      tls_client.NewClient(tls_client.NewClientOptions(300, common.GetClientProfile())),
		Auth:      &chatRequirements{},
		BaseURL:   baseURL,
		ChatURL:   baseURL + "/backend-anon/conversation",
		UserAgent: common.GetUa(),
		SessionID: uuid.New().String(),
		DeviceID:  deviceID,
	}
	if c.HTTP == nil {
		return nil, fmt.Errorf("http client is nil")
	}
	if strings.HasPrefix(token, "Bearer ") {
		c.AccAuth = token
		c.ChatURL = baseURL + "/backend-api/conversation"
	}
	proxy := strings.TrimSpace(accountProxy)
	if proxy == "" {
		proxy = strings.TrimSpace(appConf.Proxy)
	}
	if proxy != "" {
		if err := c.HTTP.SetProxy(proxy); err != nil {
			return nil, err
		}
	}
	c.loadPowResources()
	if err := c.loadRequirements(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) Headers(url string) (tls_client_httpi.Headers, tls_client_httpi.Cookies) {
	headers := tls_client_httpi.Headers{}
	headers.Set("accept", "*/*")
	headers.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	headers.Set("origin", c.BaseURL)
	headers.Set("referer", c.BaseURL+"/")
	headers.Set("priority", "u=1, i")
	headers.Set("sec-ch-ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Microsoft Edge";v="146"`)
	headers.Set("sec-ch-ua-arch", `"x86"`)
	headers.Set("sec-ch-ua-bitness", `"64"`)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-model", `""`)
	headers.Set("sec-ch-ua-platform", `"Windows"`)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	headers.Set("user-agent", c.UserAgent)
	headers.Set("oai-device-id", c.DeviceID)
	headers.Set("oai-session-id", c.SessionID)
	headers.Set("oai-language", "zh-CN")
	headers.Set("oai-client-version", "prod-81e0c5cdf6140e8c5db714d613337f4aeab94029")
	headers.Set("oai-client-build-number", "6128297")
	if c.AccAuth != "" {
		headers.Set("authorization", c.AccAuth)
	}
	return headers, c.Cookies
}

func (c *Client) IsAuthenticated() bool {
	return c.AccAuth != ""
}

func (c *Client) ChatTimezone() (string, int) {
	if c.IsAuthenticated() {
		return "Asia/Shanghai", -480
	}
	return "America/Los_Angeles", 480
}

func (c *Client) loadPowResources() {
	headers, cookies := c.Headers(c.BaseURL + "/")
	headers.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	headers.Set("sec-fetch-dest", "document")
	headers.Set("sec-fetch-mode", "navigate")
	headers.Set("sec-fetch-site", "none")
	headers.Set("sec-fetch-user", "?1")
	headers.Set("upgrade-insecure-requests", "1")
	response, err := c.HTTP.Request(tls_client_httpi.GET, c.BaseURL+"/", headers, cookies, nil)
	if err != nil {
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return
	}
	c.Pow = ParseResources(string(body))
}

func (c *Client) loadRequirements() error {
	requirementsToken := LegacyRequirementsToken(c.UserAgent, c.DeviceID, c.Pow)
	prepare, err := c.sentinelPrepare(requirementsToken)
	if err != nil {
		return err
	}
	if prepare.ForceLogin {
		common.SubUpdateThreshold()
		return fmt.Errorf("force login required")
	}
	if prepare.Arkose.Required {
		return fmt.Errorf("arkose token is required")
	}
	var proofToken string
	if prepare.ProofWork.Required {
		proofToken = CalcProofToken(prepare.ProofWork.Seed, prepare.ProofWork.Difficulty, c.UserAgent, c.DeviceID, c.Pow)
		if proofToken == "" {
			return fmt.Errorf("proof token calculation failed")
		}
	}
	var turnstileToken string
	if prepare.Turnstile.Required && prepare.Turnstile.Dx != "" {
		sourceP := ""
		if c.AccAuth == "" {
			sourceP = requirementsToken
		}
		turnstileToken = Solve(prepare.Turnstile.Dx, sourceP)
		if turnstileToken == "" {
			fallbackP := requirementsToken
			if sourceP == requirementsToken {
				fallbackP = ""
			}
			turnstileToken = Solve(prepare.Turnstile.Dx, fallbackP)
		}
	}
	finalize, err := c.sentinelFinalize(prepare.PrepareToken, proofToken, turnstileToken)
	if err != nil {
		return err
	}
	if finalize.Token == "" {
		return fmt.Errorf("missing finalized sentinel token")
	}
	c.Auth.Token = finalize.Token
	c.Auth.PrepareToken = prepare.PrepareToken
	c.Auth.ProofWork.Ospt = proofToken
	c.Auth.TurnstileToken = turnstileToken
	return nil
}

func (c *Client) sentinelPrepare(requirementsToken string) (*chatRequirements, error) {
	path := "/sentinel/chat-requirements/prepare"
	authURL := c.BaseURL + "/backend-anon" + path
	targetPath := "/backend-anon" + path
	if c.AccAuth != "" {
		authURL = c.BaseURL + "/backend-api" + path
		targetPath = "/backend-api" + path
	}
	bodyJSON, err := json.Marshal(map[string]string{"p": requirementsToken})
	if err != nil {
		return nil, err
	}
	headers, cookies := c.Headers(authURL)
	headers.Set("content-type", "application/json")
	headers.Set("x-openai-target-path", targetPath)
	headers.Set("x-openai-target-route", targetPath)
	if c.AccAuth == "" {
		headers.Set("oai-device-id", c.DeviceID)
	}
	response, err := c.HTTP.Request(tls_client_httpi.POST, authURL, headers, cookies, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("sentinel prepare failed: status=%d body=%s", response.StatusCode, string(detail))
	}
	var result chatRequirements
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

type sentinelFinalizeResponse struct {
	Persona     string `json:"persona,omitempty"`
	Token       string `json:"token"`
	ExpireAfter int    `json:"expire_after,omitempty"`
}

func (c *Client) sentinelFinalize(prepareToken, proofToken, turnstileToken string) (*sentinelFinalizeResponse, error) {
	path := "/sentinel/chat-requirements/finalize"
	authURL := c.BaseURL + "/backend-anon" + path
	targetPath := "/backend-anon" + path
	if c.AccAuth != "" {
		authURL = c.BaseURL + "/backend-api" + path
		targetPath = "/backend-api" + path
	}
	payload := map[string]string{"prepare_token": prepareToken}
	if proofToken != "" {
		payload["proofofwork"] = proofToken
	}
	if turnstileToken != "" {
		payload["turnstile"] = turnstileToken
	}
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers, cookies := c.Headers(authURL)
	headers.Set("content-type", "application/json")
	headers.Set("x-openai-target-path", targetPath)
	headers.Set("x-openai-target-route", targetPath)
	if c.AccAuth == "" {
		headers.Set("oai-device-id", c.DeviceID)
	}
	response, err := c.HTTP.Request(tls_client_httpi.POST, authURL, headers, cookies, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("sentinel finalize failed: status=%d body=%s", response.StatusCode, string(detail))
	}
	var result sentinelFinalizeResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func Retry() int {
	return constant.ReTry
}
