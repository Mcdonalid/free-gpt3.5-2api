package env

import (
	"os"
	"strings"
)

type Env string

const (
	DEV  Env = "dev"
	TEST Env = "test"
	PROD Env = "prod"
)

var Curr = Env(func() string {
	env := strings.ToLower(os.Getenv("ENV"))
	if env == "" {
		env = "dev"
	}
	return env
}())
