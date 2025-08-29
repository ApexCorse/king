package discord

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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
}

func (b *DiscordBot) onPushWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Received push event for repository: %s", body)

	payload := &PushEvent{}
	if err := json.Unmarshal(body, payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := formatPushEvent(payload)
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

func formatPushEvent(payload *PushEvent) string {
	msg := "🚀 **New push in repository**: `" + payload.Repository.Name + "`\n"
	msg += "🌿 **Branch**: `" + payload.Ref + "`\n"
	msg += "👤 **Author**: `" + payload.Pusher.Name + "`\n"
	if payload.Forced {
		msg += "⚠️ **Force Push**: `Yes`\n"
	} else {
		msg += "✅ **Force Push**: `No`\n"
	}
	msg += "\n"
	if len(payload.Commits) == 0 {
		msg += "📝 **No commits in this push.**"
	} else {
		msg += "📝 **Commits**:\n"
		for i, commit := range payload.Commits {
			msg +=
				"  " +
					"🔸 [" + commit.Message + "](" + commit.URL + ")\n" +
					"     ✍️ " + commit.Author.Name + "\n"
			if i < len(payload.Commits)-1 {
				msg += "\n"
			}
		}
	}
	return msg
}
