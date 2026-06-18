package risk

import "strings"

type Result struct {
	Level        string
	ApprovalType string
	Reason       string
	Keyword      string
}

type rule struct {
	keyword      string
	level        string
	approvalType string
}

var rules = []rule{
	{"private key", "critical", "access_sensitive_data"},
	{"seed phrase", "critical", "access_sensitive_data"},
	{"wallet", "critical", "access_sensitive_data"},
	{"funds", "critical", "access_sensitive_data"},
	{"私钥", "critical", "access_sensitive_data"},
	{"钱包", "critical", "access_sensitive_data"},
	{"资金", "critical", "access_sensitive_data"},
	{"交易", "critical", "enable_live_trading"},
	{"live trading", "critical", "enable_live_trading"},
	{"risk rule", "critical", "change_risk_rule"},
	{"风控", "critical", "change_risk_rule"},
	{"production", "high", "deploy_production"},
	{"生产", "high", "deploy_production"},
	{"deploy", "high", "deploy_production"},
	{"部署", "high", "deploy_production"},
	{"上线", "high", "deploy_production"},
	{"merge", "high", "merge_code"},
	{"合并", "high", "merge_code"},
	{"publish", "high", "publish_content"},
	{"发布", "high", "publish_content"},
	{"announcement", "high", "send_telegram_announcement"},
	{"公告", "high", "send_telegram_announcement"},
	{"KOL", "high", "contact_kol"},
	{"外部工具", "high", "connect_external_tool"},
	{"路线图", "high", "modify_project_roadmap"},
}

func Detect(text string) Result {
	lower := strings.ToLower(text)
	for _, r := range rules {
		needle := strings.ToLower(r.keyword)
		if strings.Contains(lower, needle) || strings.Contains(text, r.keyword) {
			return Result{Level: r.level, ApprovalType: r.approvalType, Keyword: r.keyword, Reason: "matched risk keyword: " + r.keyword}
		}
	}
	return Result{Level: "low", Reason: "no high-risk keyword detected"}
}

func NeedsApproval(level string) bool { return level == "high" || level == "critical" }
