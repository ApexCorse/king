package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Formula-SAE/discord/internal/utils"
)

type Author struct {
	Name string `json:"name"`
}

type PushEvent struct {
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Ref     string `json:"ref"`
	Pusher  Author `json:"pusher"`
	Forced  bool   `json:"forced"`
	Commits []struct {
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
		Author    Author `json:"author"`
	} `json:"commits"`
	Deleted bool `json:"deleted"`
	Created bool `json:"created"`
}

func (b *DiscordBot) onPushWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload := &PushEvent{}
	if err := json.Unmarshal(body, payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Received push event:\n%+v", payload)

	var msg string
	if payload.Deleted {
		msg = formatBranchDeleted(payload)
	} else if len(payload.Commits) == 0 {
		msg = formatBranchCreated(payload)
	} else {
		msg = formatStandardPush(payload)
	}

	log.Printf("Formatted push event: %s", msg)

	subscriptions, err := b.db.GetWebhookSubscriptionsByRepository(payload.Repository.Name)
	if err != nil {
		log.Printf("Error getting webhook subscriptions: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		_, err := b.session.ChannelMessageSend(subscription.ChannelID, msg)
		if err != nil {
			log.Printf("Error sending message to channel %s: %v", subscription.ChannelID, err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func formatStandardPush(payload *PushEvent) string {
	msg := fmt.Sprintf("🚀 %s: %s\n", utils.Bold("New push in repository"), utils.InlineCode(payload.Repository.Name))
	branchLine := fmt.Sprintf("🌿 %s: %s", utils.Bold("Branch"), utils.InlineCode(getBranchName(payload.Ref)))
	if payload.Created {
		branchLine += fmt.Sprintf(" (🆕 %s)\n", utils.Italic("NEW BRANCH"))
	} else {
		branchLine += "\n"
	}
	msg += branchLine
	msg += fmt.Sprintf("👤 %s: %s\n", utils.Bold("Author"), utils.InlineCode(payload.Pusher.Name))
	if payload.Forced {
		msg += fmt.Sprintf("⚠️ %s\n", utils.Bold("FORCED PUSH"))
	}
	msg += "\n"

	msg += fmt.Sprintf("📝 %s:\n", utils.Bold("Commits"))
	for i, commit := range payload.Commits {
		msg += getCommitMessage(
			commit.Message,
			commit.URL,
			commit.Author.Name,
		)
		if i < len(payload.Commits)-1 {
			msg += "\n"
		}
	}
	return msg
}

func formatBranchCreated(payload *PushEvent) string {
	msg := fmt.Sprintf("🚀 %s: %s\n", utils.Bold("New branch created"), utils.InlineCode(payload.Repository.Name))
	msg += fmt.Sprintf("🌿 %s: %s\n", utils.Bold("Branch"), utils.InlineCode(getBranchName(payload.Ref)))
	msg += fmt.Sprintf("👤 %s: %s\n", utils.Bold("Author"), utils.InlineCode(payload.Pusher.Name))
	return msg
}

func formatBranchDeleted(payload *PushEvent) string {
	msg := fmt.Sprintf("🗑️ %s: %s\n", utils.Bold("Branch deleted"), utils.InlineCode(payload.Repository.Name))
	msg += fmt.Sprintf("🌿 %s: %s\n", utils.Bold("Branch"), utils.InlineCode(getBranchName(payload.Ref)))
	msg += fmt.Sprintf("👤 %s: %s\n", utils.Bold("Author"), utils.InlineCode(payload.Pusher.Name))
	return msg
}

func getBranchName(ref string) string {
	if after, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		return after
	}
	return ref
}

func getCommitMessage(commit string, link string, author string) string {
	parts := strings.SplitN(commit, "\n\n", 2)
	if len(parts) == 0 {
		return ""
	}
	title := parts[0]

	msg := fmt.Sprintf("  🔸 %s\n", utils.Link(title, link))
	msg += fmt.Sprintf("     ✍️ %s", author)

	if len(parts) == 2 {
		msg += "\n\n" + parts[1]
	}

	return msg
}
