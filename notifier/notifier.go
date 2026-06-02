// Package notifier defines the Notifier interface and built-in implementations
// for sending probe state-change alerts to external receivers.
package notifier

// Notifier is implemented by any alert receiver.
// Implementations must be safe for concurrent use.
type Notifier interface {
	// Alert is called when a probe changes state.
	// up is true on recovery (down→up) and false on first failure or (up→down).
	Alert(name, targetURL string, up bool, errMsg string)
}

type multi struct {
	notifiers []Notifier
}

// Multi returns a Notifier that fans out every alert to each of nn in order.
// If nn is empty, alerts are silently dropped.
func Multi(nn ...Notifier) Notifier {
	return &multi{notifiers: nn}
}

func (m *multi) Alert(name, targetURL string, up bool, errMsg string) {
	for _, n := range m.notifiers {
		n.Alert(name, targetURL, up, errMsg)
	}
}
