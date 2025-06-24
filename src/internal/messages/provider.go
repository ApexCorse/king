package messages

type Provider interface {
	SendMessage(configs ...MessageConfig) error
}

type ProviderGroup struct {
	providers []Provider
}

func NewProviderGroup(providers ...Provider) *ProviderGroup {
	return &ProviderGroup{providers: providers}
}

func (g *ProviderGroup) SendMessage(configs ...MessageConfig) {
	for _, p := range g.providers {
		if p == nil {
			continue
		}

		p.SendMessage(configs...)
	}
}
