package llm

import (
	"regexp"
	"strings"
)

var redactionRules = []*regexp.Regexp{
	regexp.MustCompile(`(?is)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`(?i)bearer\s+[a-z0-9._~+/=-]{12,}`),
	regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|private[_-]?key)\s*[:=]\s*[^\s,;]+`),
	regexp.MustCompile(`(?i)\b(sk-[a-z0-9_-]{12,}|deepseek-[a-z0-9_-]{12,})\b`),
	regexp.MustCompile(`\b0x[a-fA-F0-9]{64}\b`),
	regexp.MustCompile(`\b[a-fA-F0-9]{64}\b`),
}

var criticalSensitiveTerms = []string{
	"private key", "private_key", "seed phrase", "mnemonic", "wallet", "secret key",
	"私钥", "助记词", "钱包", "资金",
}

func SanitizeText(text string) string {
	out := text
	for _, rule := range redactionRules {
		out = rule.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}

func ContainsCriticalSensitiveText(text string) bool {
	lower := strings.ToLower(text)
	for _, term := range criticalSensitiveTerms {
		if strings.Contains(lower, strings.ToLower(term)) || strings.Contains(text, term) {
			return true
		}
	}
	return false
}
