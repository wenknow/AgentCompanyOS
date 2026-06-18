package telegram

import (
	"context"
	"strconv"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/app"
	"github.com/agentcompany/agent-company-os/backend/internal/command"
	"github.com/agentcompany/agent-company-os/backend/internal/workflow"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	services *app.Services
	store    MessageStore
	log      *zap.Logger
}

func NewBot(token string, services *app.Services, store MessageStore, log *zap.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	if _, err := api.Request(tgbotapi.DeleteWebhookConfig{}); err != nil {
		return nil, err
	}
	if _, err := api.Request(tgbotapi.NewSetMyCommands(botCommands()...)); err != nil {
		return nil, err
	}
	return &Bot{api: api, services: services, store: store, log: log}, nil
}

func botCommands() []tgbotapi.BotCommand {
	return []tgbotapi.BotCommand{
		{Command: "start", Description: "Start AgentCompanyOS"},
		{Command: "help", Description: "Show available commands"},
		{Command: "status", Description: "Show company status"},
		{Command: "agents", Description: "List built-in agents"},
		{Command: "project", Description: "Manage projects"},
		{Command: "projects", Description: "List projects"},
		{Command: "assign", Description: "Assign a task to an agent"},
		{Command: "task", Description: "Manage tasks"},
		{Command: "tasks", Description: "List recent tasks"},
		{Command: "feedback", Description: "Record project feedback"},
		{Command: "autopilot", Description: "Run project autopilot"},
		{Command: "deploy", Description: "Request project deployment"},
		{Command: "approvals", Description: "List pending approvals"},
		{Command: "approve", Description: "Approve an item"},
		{Command: "reject", Description: "Reject an item"},
		{Command: "daily", Description: "Generate daily report"},
		{Command: "weekly", Description: "Generate weekly report"},
		{Command: "plan", Description: "Create product and technical plan"},
		{Command: "build", Description: "Create coding workflow draft"},
		{Command: "launch", Description: "Create launch draft"},
		{Command: "review", Description: "Create QA or compliance review"},
		{Command: "runs", Description: "List agent runs"},
		{Command: "artifacts", Description: "List artifacts"},
		{Command: "runtime", Description: "Show runtime status"},
	}
}

func (b *Bot) Run(ctx context.Context) error {
	config := tgbotapi.NewUpdate(0)
	config.Timeout = 30
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := b.api.GetUpdates(config)
		if err != nil {
			b.log.Warn("telegram get updates failed", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= config.Offset {
				config.Offset = update.UpdateID + 1
			}
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	cmd := command.Parse(update.Message.Text, update.Message.Chat.ID, update.Message.From.ID)
	b.log.Info("telegram command received", zap.String("command", cmd.Name), zap.Int64("user_id", cmd.UserID), zap.Int64("chat_id", cmd.ChatID))
	if err := b.store.Record(ctx, cmd); err != nil {
		b.log.Warn("record telegram message failed", zap.Error(err))
	}
	if !b.services.Config.UserAllowed(cmd.UserID) {
		b.reply(update.Message.Chat.ID, "当前用户没有使用此 bot 的权限。")
		return
	}
	actor := "telegram:" + strconv.FormatInt(cmd.UserID, 10)
	if isAsyncCommand(cmd.Name) {
		chatID := update.Message.Chat.ID
		commandName := cmd.Name
		args := append([]string(nil), cmd.Args...)
		b.reply(chatID, "已开始执行 /"+commandName+"。我会持续汇报关键阶段，完成后发送结果。")
		go func() {
			progressCtx := workflow.WithProgress(context.Background(), func(message string) {
				b.reply(chatID, message)
			})
			resp, err := b.services.HandleCommand(progressCtx, commandName, args, actor)
			if err != nil {
				b.log.Warn("async command failed", zap.String("command", commandName), zap.Error(err))
				resp = "命令执行失败，请查看 bot 日志或稍后重试。"
			}
			b.reply(chatID, resp)
		}()
		return
	}
	resp, err := b.services.HandleCommand(ctx, cmd.Name, cmd.Args, actor)
	if err != nil {
		b.log.Warn("command failed", zap.String("command", cmd.Name), zap.Error(err))
		resp = "命令执行失败，请查看 bot 日志或稍后重试。"
	}
	b.reply(update.Message.Chat.ID, resp)
}

func isAsyncCommand(name string) bool {
	switch name {
	case "plan", "build", "launch", "review", "autopilot":
		return true
	default:
		return false
	}
}

func (b *Bot) reply(chatID int64, text string) {
	if text == "" {
		text = "完成。"
	}
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.log.Warn("telegram send message failed", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}
