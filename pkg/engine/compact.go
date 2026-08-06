package engine

import "github.com/mirkobrombin/nan-core/pkg/store"

// CompactToWAL writes a canonical WAL from the current engine snapshot.
func (e *Engine) CompactToWAL(path string) error {
	w, err := store.CreateWAL(path)
	if err != nil {
		return err
	}
	defer w.Close()

	for _, b := range e.CanonicalBeliefs() {
		payload, err := encodeBelief(b)
		if err != nil {
			return err
		}
		if err := w.Append(store.Record{Type: store.EventFactAdded, Payload: payload}); err != nil {
			return err
		}
	}
	for _, r := range e.CanonicalResolutions() {
		payload, err := encodeResolution(r)
		if err != nil {
			return err
		}
		if err := w.Append(store.Record{Type: store.EventResolutionSet, Payload: payload}); err != nil {
			return err
		}
	}
	for _, rule := range e.RulesSnapshot() {
		payload, err := encodeRule(rule)
		if err != nil {
			return err
		}
		if err := w.Append(store.Record{Type: store.EventRuleAdded, Payload: payload}); err != nil {
			return err
		}
	}
	return nil
}
