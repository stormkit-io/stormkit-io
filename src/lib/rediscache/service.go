package rediscache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/lib/shutdown"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

const (
	ServiceWorkerserver = "workerserver"
	ServiceHosting      = "hosting"
	ServiceApi          = "api"
)

const (
	StatusSent       = "sent"
	StatusProcessing = "processing"
	StatusOK         = "ok"
	StatusErr        = "error"
)

const (
	EventInvalidateAdminCache   = "invalidate_admin_cache"
	EventInvalidateHostingCache = "cache_invalidate"
	EventMiseUpdate             = "mise_update"
	EventRuntimesInstall        = "runtimes_install"
)

const (
	KEY_RUNTIMES_STATUS = "runtimes_status"
)

type Handler func(context.Context, ...string)

type MicroService struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	ctx    context.Context
	cancel context.CancelFunc
	client *RedisCache
}

type MicroServiceInterface interface {
	Key(string) string
	Subscribe(string, Handler) error
	SubscribeAsync(string, Handler) error
	Broadcast(string, ...string) error
	List([]string) ([]*MicroService, error)
	SetAll(string, string, []string) error
	GetAll(string, []string) (map[string]string, error)
	DelAll(string, []string) error
}

var _service MicroServiceInterface
var _smux sync.Mutex

var DefaultService MicroServiceInterface

func newService() MicroServiceInterface {
	ctx, cancel := context.WithCancel(context.Background())
	client := Client()

	// If the Redis client is not available, we return a service with no ID and name.
	if client == nil {
		return &MicroService{}
	}

	service := &MicroService{
		ID:     uuid.New().String(),
		Name:   utils.GetString(os.Getenv("STORMKIT_SERVICE_NAME"), "no-name"),
		ctx:    ctx,
		cancel: cancel,
		client: client,
	}

	key := service.Key("sd")
	data, _ := json.Marshal(service)

	slog.Debug(slog.LogOpts{
		Msg:   "registering service in service discovery",
		Level: slog.DL1,
		Payload: []zap.Field{
			zap.String("id", service.ID),
			zap.String("name", service.Name),
			zap.String("key", key),
		},
	})

	client.Set(service.ctx, key, data, 0)

	shutdown.Subscribe(func() error {

		slog.Debug(slog.LogOpts{
			Msg:   "removing service from service discovery",
			Level: slog.DL1,
			Payload: []zap.Field{
				zap.String("id", service.ID),
				zap.String("name", service.Name),
				zap.String("key", key),
			},
		})

		service.cancel()

		if err := client.Del(context.Background(), key).Err(); err != nil {
			slog.Errorf("error while removing service %s from service discovery: %v", service.ID, err)
		}

		return nil
	})

	return service
}

// Service returns a singleton instance of MicroService.
// It initializes the service if it does not exist yet, ensuring thread safety with a mutex.
// The service is registered in Redis with a unique ID and name.
func Service() MicroServiceInterface {
	_smux.Lock()
	defer _smux.Unlock()

	if DefaultService != nil {
		return DefaultService
	}

	if _service == nil {
		_service = newService()
	}

	return _service
}

// Key generates a unique key for the microservice based on its name and ID.
func (s *MicroService) Key(name string) string {
	return fmt.Sprintf("service:%s:%s:%s", name, s.Name, s.ID)
}

// Subscribe registers a new event handler for the microservice
func (s *MicroService) Subscribe(event string, handler Handler) error {
	if s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	sub := s.client.Subscribe(s.ctx, event)
	chn := sub.Channel()
	ctx := context.Background()

	for {
		select {
		case <-s.ctx.Done():
			return sub.Close()
		case msg, ok := <-chn:
			if !ok {
				break
			}

			handler(ctx, msg.Payload)
		}
	}
}

// SubscribeAsync registers a new event handler for the microservice asynchronously.
func (s *MicroService) SubscribeAsync(event string, handler Handler) error {
	if s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	go s.Subscribe(event, handler)

	return nil
}

// Broadcast sends a message to all subscribers of the specified event.
func (s *MicroService) Broadcast(event string, payload ...string) error {
	p := ""

	if len(payload) > 0 {
		p = payload[0]
	}

	return Client().Publish(s.ctx, event, p).Err()
}

// List retrieves all registered microservices from Redis.
func (s *MicroService) List(filter []string) ([]*MicroService, error) {
	client := Client()

	keys, err := client.Keys(s.ctx, "service:sd:*")

	if err != nil {
		return nil, err
	}

	var services []*MicroService

	for _, key := range keys {
		data, err := client.Get(s.ctx, key).Result()

		if err != nil {
			continue
		}

		var service MicroService

		if err := json.Unmarshal([]byte(data), &service); err != nil {
			continue
		}

		if len(filter) == 0 || utils.InSliceString(filter, service.Name) {
			services = append(services, &service)
		}
	}

	return services, nil
}

// SetAll sets the specified key to the given value for all registered microservices.
func (s *MicroService) SetAll(key, value string, filter []string) error {
	services, err := s.List(filter)

	if err != nil {
		return err
	}

	// Set the status for the runtimes in each service
	for _, service := range services {
		slog.Debug(slog.LogOpts{
			Msg:   "setting key for service",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("service_id", service.ID),
				zap.String("service_name", service.Name),
				zap.String("key", key),
				zap.String("value", value),
			},
		})

		s.client.Set(s.ctx, service.Key(key), value, time.Hour)
	}

	return nil
}

// GetAll retrieves the value of the specified key from all registered microservices.
func (s *MicroService) GetAll(key string, filter []string) (map[string]string, error) {
	services, err := s.List(filter)

	if err != nil {
		return nil, err
	}

	result := make(map[string]string)

	for _, service := range services {
		value, err := s.client.Get(s.ctx, service.Key(key)).Result()
		result[service.ID] = ""

		if err != nil {
			continue
		}

		result[service.ID] = value
	}

	return result, nil
}

// DelAll deletes the specified key from all registered microservices.
func (s *MicroService) DelAll(key string, filter []string) error {
	services, err := s.List(filter)

	if err != nil {
		return err
	}

	for _, service := range services {
		s.client.Del(s.ctx, service.Key(key))
	}

	return nil
}

// Status checks the status of the specified key across all registered microservices.
// It returns StatusOK only if all services have StatusOK, otherwise it returns the
// first encountered status.
func Status(ctx context.Context, key string, filter []string) (string, error) {
	status, err := Service().GetAll(key, filter)

	if err != nil {
		return "", err
	}

	for _, v := range status {
		if v == StatusErr {
			return StatusErr, nil
		}

		if v == StatusProcessing {
			return StatusProcessing, nil
		}

		if v == StatusSent {
			return StatusSent, nil
		}
	}

	return StatusOK, nil
}

// Broadcast is a convenience function to broadcast an event with optional payload.
func Broadcast(event string, payload ...string) error {
	return Service().Broadcast(event, payload...)
}

// SetAll is a convenience function to set a key-value pair across all services with optional filtering.
func SetAll(key, value string, filter []string) error {
	return Service().SetAll(key, value, filter)
}

// GetAll is a convenience function to get the values of a key across all services with optional filtering.
func GetAll(key string, filter []string) (map[string]string, error) {
	return Service().GetAll(key, filter)
}

// DelAll is a convenience function to delete a key across all services with optional filtering.
func DelAll(key string, filter []string) error {
	return Service().DelAll(key, filter)
}
