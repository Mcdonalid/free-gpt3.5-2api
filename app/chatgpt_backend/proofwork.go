package chatgpt_backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPowScript  = "https://chatgpt.com/backend-api/sentinel/sdk.js"
	sentinelSDKScript = "https://chatgpt.com/sentinel/20260423af3c/sdk.js"
	powMaxAttempts    = 500000
	powFailPrefix     = "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D"
)

var (
	screenSizes  = [][2]int{{1920, 1080}, {2560, 1440}, {1536, 864}, {1440, 900}, {1366, 768}}
	documentKeys = []string{
		"_reactListening8in7sfyhjvp", "_reactListeningo743lnnpvdg",
		"_reactContainer$5pyziap1brc", "__reactContainer$b63yiita51i",
		"location", "cookie", "referrer", "currentScript", "body", "head", "documentElement",
	}
	navigatorKeys = []string{
		"windowControlsOverlay−[object WindowControlsOverlay]",
		"geolocation−[object Geolocation]",
		"clipboard−[object Clipboard]",
		"mediaDevices−[object MediaDevices]",
		"permissions−[object Permissions]",
		"bluetooth−[object Bluetooth]",
		"usb−[object USB]",
		"serial−[object Serial]",
		"hid−[object HID]",
		"presentation−[object Presentation]",
		"credentials−[object CredentialsContainer]",
	}
	windowKeys = []string{
		"onchange", "onclick", "onload", "onerror", "onresize",
		"onmouseover", "onmouseout", "onfocus", "onblur", "onscroll",
		"onkeydown", "onkeyup", "onkeypress",
		"requestIdleCallback", "requestAnimationFrame", "setTimeout",
		"fetch", "console", "Promise", "Map", "Set", "WeakMap", "WeakSet",
		"crypto", "performance", "navigator", "document", "location", "history",
		"localStorage", "sessionStorage", "indexedDB",
		"Image", "XMLHttpRequest", "FormData", "Headers", "Request", "Response",
		"alert", "confirm", "prompt", "close", "focus", "blur",
		"addEventListener", "removeEventListener", "dispatchEvent",
		"scrollTo", "scrollBy", "scroll", "matchMedia", "getComputedStyle",
		"getSelection", "find", "stop", "open", "print", "captureEvents",
		"releaseEvents", "queueMicrotask", "reportError", "structuredClone",
		"isSecureContext", "crossOriginIsolated", "originAgentCluster",
		"speechSynthesis", "MediaSource", "Blob", "File", "FileReader",
		"Atomics", "SharedArrayBuffer", "WebAssembly", "BigInt", "Symbol", "Proxy",
	}
	scriptSrcRE = regexp.MustCompile(`<script\b[^>]*\bsrc=["']([^"']+)["']`)
	dataBuildRE = regexp.MustCompile(`(?:c/[^/]*/_|<html[^>]*data-build=["']([^"']*)["'])`)
)

type ProofWork struct {
	Difficulty string `json:"difficulty,omitempty"`
	Required   bool   `json:"required"`
	Seed       string `json:"seed,omitempty"`
	Ospt       string `json:"-"`
}

type Resources struct {
	ScriptSources []string
	DataBuild     string
}

func ParseResources(html string) Resources {
	resources := Resources{}
	for _, match := range scriptSrcRE.FindAllStringSubmatch(html, -1) {
		resources.ScriptSources = append(resources.ScriptSources, match[1])
		if resources.DataBuild == "" {
			if build := regexp.MustCompile(`c/[^/]*/_`).FindString(match[1]); build != "" {
				resources.DataBuild = build
			}
		}
	}
	if len(resources.ScriptSources) == 0 {
		resources.ScriptSources = []string{defaultPowScript}
	}
	if resources.DataBuild == "" {
		for _, match := range dataBuildRE.FindAllStringSubmatch(html, -1) {
			if len(match) > 1 && match[1] != "" {
				resources.DataBuild = match[1]
				break
			}
			if match[0] != "" && strings.HasPrefix(match[0], "c/") {
				resources.DataBuild = match[0]
				break
			}
		}
	}
	return resources
}

func CalcProofToken(seed string, difficulty string, userAgent string, deviceID string, resources ...Resources) string {
	if seed == "" || difficulty == "" {
		return "gAAAAAB~S"
	}
	start := time.Now()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	powConfig := fingerprintConfig{
		UserAgent: userAgent,
		DeviceID:  deviceID,
		Resources: firstResource(resources),
	}
	for i := 0; i < powMaxAttempts; i++ {
		elapsed := time.Since(start).Milliseconds()
		config := powConfig.build(rng, &i, &elapsed)
		encoded := encodeConfig(config)
		hashResult := fnv1aHash(seed + encoded)
		if len(difficulty) > len(hashResult) {
			continue
		}
		if hashResult[:len(difficulty)] <= difficulty {
			return "gAAAAAB" + encoded + "~S"
		}
	}
	return "gAAAAAB" + powFailPrefix + base64.StdEncoding.EncodeToString([]byte(`"e"`)) + "~S"
}

func LegacyRequirementsToken(userAgent string, deviceID string, resources ...Resources) string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	config := fingerprintConfig{
		UserAgent: userAgent,
		DeviceID:  deviceID,
		Resources: firstResource(resources),
	}.build(rng, nil, nil)
	return "gAAAAAC" + encodeConfig(config) + "~S"
}

func firstResource(resources []Resources) Resources {
	if len(resources) > 0 {
		return resources[0]
	}
	return Resources{ScriptSources: []string{defaultPowScript}}
}

type fingerprintConfig struct {
	UserAgent string
	DeviceID  string
	Resources Resources
}

func (c fingerprintConfig) build(rng *rand.Rand, nonce *int, elapsedMs *int64) []interface{} {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	resources := c.Resources
	scriptSources := append([]string{}, resources.ScriptSources...)
	if len(scriptSources) == 0 {
		scriptSources = []string{defaultPowScript}
	}
	scriptSources = append(scriptSources, sentinelSDKScript)
	screen := screenSizes[rng.Intn(len(screenSizes))]
	perfNow := rng.Float64() * 10000
	timeOrigin := float64(time.Now().UnixMilli()) - perfNow
	nonceValue := 1
	if nonce != nil {
		nonceValue = *nonce
	}
	randomOrElapsed := rng.Float64()
	if elapsedMs != nil {
		randomOrElapsed = float64(*elapsedMs)
	}
	userAgent := c.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
	}
	deviceID := c.DeviceID
	if deviceID == "" {
		deviceID = uuid.NewString()
	}
	return []interface{}{
		screen[0] + screen[1],
		jsDateString(time.Now(), "America/Los_Angeles"),
		int64(4294967296),
		nonceValue,
		userAgent,
		scriptSources[rng.Intn(len(scriptSources))],
		resources.DataBuild,
		"en-US",
		"en-US,en",
		randomOrElapsed,
		navigatorKeys[rng.Intn(len(navigatorKeys))],
		documentKeys[rng.Intn(len(documentKeys))],
		windowKeys[rng.Intn(len(windowKeys))],
		perfNow,
		deviceID,
		"",
		8 + rng.Intn(4)*4,
		timeOrigin,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
	}
}

func jsDateString(t time.Time, timezone string) string {
	if loc, err := time.LoadLocation(timezone); err == nil {
		t = t.In(loc)
	}
	head := t.Format("Mon Jan 2 2006 15:04:05")
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	name, _ := t.Zone()
	full := map[string]string{
		"PDT": "Pacific Daylight Time",
		"PST": "Pacific Standard Time",
		"EDT": "Eastern Daylight Time",
		"EST": "Eastern Standard Time",
	}[name]
	if full == "" {
		full = name
	}
	return fmt.Sprintf("%s GMT%s%02d%02d (%s)", head, sign, hours, minutes, full)
}

func encodeConfig(config []interface{}) string {
	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func fnv1aHash(text string) string {
	const (
		fnvOffset = 2166136261
		fnvPrime  = 16777619
	)
	h := uint32(fnvOffset)
	for _, ch := range text {
		h ^= uint32(ch)
		h *= fnvPrime
	}
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return fmt.Sprintf("%08x", h)
}
