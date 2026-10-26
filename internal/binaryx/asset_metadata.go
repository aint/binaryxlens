package binaryx

const (
	Bali       Location = "bali"
	Turkey     Location = "turkey"
	Montenegro Location = "montenegro"
)

type Location string

func (l Location) Flag() string {
	switch l {
	case Bali:
		return "🇮🇩"
	case Turkey:
		return "🇹🇷"
	case Montenegro:
		return "🇲🇪"
	default:
		return ""
	}
}
