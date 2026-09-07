package deploy_test

import (
	"encoding/json"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
)

// fakeService stands in for service discovery and the event bus, so a test can
// say which hosting nodes exist and answer the warm-up itself.
type fakeService struct {
	id          string
	nodes       []string
	onBroadcast func(warmupID string)
}

func (f *fakeService) ServiceID() string { return f.id }

func (f *fakeService) Key(name string) string {
	return "service:" + name + ":test:" + f.id
}

func (f *fakeService) List(filter []string) ([]*rediscache.MicroService, error) {
	services := make([]*rediscache.MicroService, 0, len(f.nodes))

	for _, id := range f.nodes {
		services = append(services, &rediscache.MicroService{ID: id, Name: rediscache.ServiceHosting})
	}

	return services, nil
}

func (f *fakeService) Broadcast(event string, payload ...string) error {
	if f.onBroadcast != nil && len(payload) > 0 {
		go f.onBroadcast(warmupIDOf(payload[0]))
	}

	return nil
}

func (f *fakeService) Subscribe(string, rediscache.Handler) error      { return nil }
func (f *fakeService) SubscribeAsync(string, rediscache.Handler) error { return nil }
func (f *fakeService) SetAll(string, string, []string) error           { return nil }
func (f *fakeService) DelAll(string, []string) error                   { return nil }

func (f *fakeService) GetAll(string, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// warmupIDOf pulls the warm-up id out of the broadcast payload.
func warmupIDOf(payload string) string {
	req := struct {
		WarmupID string `json:"warmupId"`
	}{}

	_ = json.Unmarshal([]byte(payload), &req)

	return req.WarmupID
}
