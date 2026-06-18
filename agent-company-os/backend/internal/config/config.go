package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                   string
	AppName                  string
	HTTPPort                 string
	DatabaseURL              string
	RedisAddr                string
	RedisPassword            string
	RedisDB                  int
	TelegramBotToken         string
	TelegramAllowedUserIDs   map[int64]struct{}
	LogLevel                 string
	DefaultProjectName       string
	LLMProvider              string
	DeepSeekAPIKey           string
	DeepSeekBaseURL          string
	DeepSeekModel            string
	DeepSeekReasoningEffort  string
	DeepSeekThinking         string
	LLMTimeoutSeconds        int
	CodingRuntime            string
	ClaudeCodeCommand        string
	ClaudeCodeTimeoutSeconds int
	ClaudeCodeWorkdir        string
	ClaudeCodeAllowedRoot    string
	ClaudeCodeMaxOutputBytes int
	ClaudeCodeEnabled        bool
	DeployAllowedRoot        string
	DeployTimeoutSeconds     int
	DeployMaxOutputBytes     int
}

func Load() Config {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	redisDB, _ := strconv.Atoi(env("REDIS_DB", "0"))
	llmTimeout, _ := strconv.Atoi(env("LLM_TIMEOUT_SECONDS", "180"))
	claudeTimeout, _ := strconv.Atoi(env("CLAUDE_CODE_TIMEOUT_SECONDS", "900"))
	claudeMaxOutput, _ := strconv.Atoi(env("CLAUDE_CODE_MAX_OUTPUT_BYTES", "200000"))
	deployTimeout, _ := strconv.Atoi(env("DEPLOY_TIMEOUT_SECONDS", "600"))
	deployMaxOutput, _ := strconv.Atoi(env("DEPLOY_MAX_OUTPUT_BYTES", "200000"))
	return Config{
		AppEnv:                   env("APP_ENV", "development"),
		AppName:                  env("APP_NAME", "agent-company-os"),
		HTTPPort:                 env("HTTP_PORT", "8080"),
		DatabaseURL:              env("DATABASE_URL", "postgres://agent:agent@localhost:5432/agent_company_os?sslmode=disable"),
		RedisAddr:                env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:            os.Getenv("REDIS_PASSWORD"),
		RedisDB:                  redisDB,
		TelegramBotToken:         os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramAllowedUserIDs:   parseAllowed(os.Getenv("TELEGRAM_ALLOWED_USER_IDS")),
		LogLevel:                 env("LOG_LEVEL", "debug"),
		DefaultProjectName:       env("DEFAULT_PROJECT_NAME", "AgentCompanyOS"),
		LLMProvider:              strings.ToLower(env("LLM_PROVIDER", "")),
		DeepSeekAPIKey:           os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL:          strings.TrimRight(env("DEEPSEEK_BASE_URL", "https://api.deepseek.com"), "/"),
		DeepSeekModel:            env("DEEPSEEK_MODEL", "deepseek-v4-pro"),
		DeepSeekReasoningEffort:  env("DEEPSEEK_REASONING_EFFORT", "high"),
		DeepSeekThinking:         strings.ToLower(env("DEEPSEEK_THINKING", "enabled")),
		LLMTimeoutSeconds:        llmTimeout,
		CodingRuntime:            strings.ToLower(env("CODING_RUNTIME", "claude_code_local")),
		ClaudeCodeCommand:        env("CLAUDE_CODE_COMMAND", "claude"),
		ClaudeCodeTimeoutSeconds: claudeTimeout,
		ClaudeCodeWorkdir:        env("CLAUDE_CODE_WORKDIR", ".."),
		ClaudeCodeAllowedRoot:    env("CLAUDE_CODE_ALLOWED_ROOT", ""),
		ClaudeCodeMaxOutputBytes: claudeMaxOutput,
		ClaudeCodeEnabled:        parseBool(env("CLAUDE_CODE_ENABLED", "false")),
		DeployAllowedRoot:        env("DEPLOY_ALLOWED_ROOT", env("CLAUDE_CODE_ALLOWED_ROOT", "")),
		DeployTimeoutSeconds:     deployTimeout,
		DeployMaxOutputBytes:     deployMaxOutput,
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseAllowed(raw string) map[int64]struct{} {
	out := map[int64]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if id, err := strconv.ParseInt(part, 10, 64); err == nil {
			out[id] = struct{}{}
		}
	}
	return out
}

func (c Config) UserAllowed(id int64) bool {
	if len(c.TelegramAllowedUserIDs) == 0 {
		return true
	}
	_, ok := c.TelegramAllowedUserIDs[id]
	return ok
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
