package discord

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
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
	msg := "🚀 **New push in repository**: `" + payload.Repository.Name + "`\n"
	msg += "🌿 **Branch**: `" + getBranchName(payload.Ref) + "`"
	if payload.Created {
		msg += "(🆕 *NEW BRANCH*)\n"
	} else {
		msg += "\n"
	}
	msg += "👤 **Author**: `" + payload.Pusher.Name + "`\n"
	if payload.Forced {
		msg += "⚠️ **FORCED PUSH**\n"
	}
	msg += "\n"

	msg += "📝 **Commits**:\n"
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
	msg := "🚀 **New branch created**: `" + payload.Repository.Name + "`\n"
	msg += "🌿 **Branch**: `" + getBranchName(payload.Ref) + "`\n"
	msg += "👤 **Author**: `" + payload.Pusher.Name + "`\n"
	return msg
}

func formatBranchDeleted(payload *PushEvent) string {
	msg := "🗑️ **Branch deleted**: `" + payload.Repository.Name + "`\n"
	msg += "🌿 **Branch**: `" + getBranchName(payload.Ref) + "`\n"
	msg += "👤 **Author**: `" + payload.Pusher.Name + "`\n"
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

	msg := "  🔸 [" + title + "](" + link + ")\n"
	msg += "     ✍️ " + author

	if len(parts) == 2 {
		msg += "\n\n" + parts[1]
	}

	return msg
}
