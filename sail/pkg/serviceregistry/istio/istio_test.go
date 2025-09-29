package istio

import (
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "istiod.istio-system.svc.cluster.local:15010", config.PilotAddress)
	assert.Equal(t, "istio-system", config.Namespace)
	assert.True(t, config.EnableDiscovery)
	assert.True(t, config.TLSEnabled)
	assert.Equal(t, 30*time.Second, config.SyncTimeout)
}

func TestNewConfigManager(t *testing.T) {
	manager := NewConfigManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
}

func TestConfigManager_LoadFromEnvironment(t *testing.T) {
	manager := NewConfigManager()

	// Test with default environment
	err := manager.LoadFromEnvironment()
	assert.NoError(t, err)

	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "istiod.istio-system.svc.cluster.local:15010", config.PilotAddress)
}

func TestConfigManager_LoadFromConfig(t *testing.T) {
	manager := NewConfigManager()

	// Test with valid config (no TLS paths to avoid file checks)
	config := &IstioConfig{
		PilotAddress:    "test.example.com:15010",
		Namespace:       "test-namespace",
		EnableDiscovery: true,
		TLSEnabled:      false, // Disable TLS to avoid cert file checks
		CertPath:        "",
		KeyPath:         "",
		CAPath:          "",
		SyncTimeout:     10 * time.Second,
	}

	err := manager.LoadFromConfig(config)
	assert.NoError(t, err)

	loadedConfig := manager.GetConfig()
	assert.Equal(t, config.PilotAddress, loadedConfig.PilotAddress)
	assert.Equal(t, config.Namespace, loadedConfig.Namespace)
	assert.Equal(t, config.TLSEnabled, loadedConfig.TLSEnabled)
}

func TestNewController(t *testing.T) {
	config := DefaultConfig()
	clusterID := cluster.ID("test-cluster")

	controller := NewController(config, clusterID)

	assert.NotNil(t, controller)
	assert.Equal(t, config, controller.config)
	assert.Equal(t, clusterID, controller.clusterID)
	assert.NotNil(t, controller.pilotClient)
	assert.NotNil(t, controller.serviceConverter)
	assert.NotNil(t, controller.configManager)
	assert.NotNil(t, controller.services)
	assert.False(t, controller.hasSynced.Load())
}

func TestController_Provider(t *testing.T) {
	config := DefaultConfig()
	clusterID := cluster.ID("test-cluster")
	controller := NewController(config, clusterID)

	providerType := controller.Provider()
	assert.Equal(t, "Istio", string(providerType))
}

func TestServiceConverter_New(t *testing.T) {
	converter := NewServiceConverter("test-cluster")

	assert.NotNil(t, converter)
	assert.Equal(t, "test-cluster", converter.clusterID)
}

func TestServiceConverter_ConvertToDubboService(t *testing.T) {
	converter := NewServiceConverter("test-cluster")

	istioService := &IstioServiceInfo{
		Hostname:  "test.example.com",
		Namespace: "default",
		Ports:     []IstioPortInfo{{Name: "http", Port: 8080, Protocol: "HTTP"}},
		Labels:    map[string]string{"app": "test"},
		VirtualService: &VirtualService{
			Name:      "test-vs",
			Namespace: "default",
			Spec:      map[string]interface{}{"weight": 100},
		},
		DestinationRule: &DestinationRule{
			Name:      "test-dr",
			Namespace: "default",
			Spec:      map[string]interface{}{"subset": "v1"},
		},
	}

	dubboService, err := converter.ConvertToDubboService(istioService)
	assert.NoError(t, err)
	assert.NotNil(t, dubboService)
	assert.Equal(t, "test.example.com", string(dubboService.Hostname))
	assert.Equal(t, "default", dubboService.Attributes.Namespace)
	assert.Len(t, dubboService.Ports, 1)
	assert.Equal(t, "http", dubboService.Ports[0].Name)
	assert.Equal(t, 8080, dubboService.Ports[0].Port)
}

func TestServiceConverter_ConvertToDubboService_Nil(t *testing.T) {
	converter := NewServiceConverter("test-cluster")

	dubboService, err := converter.ConvertToDubboService(nil)
	assert.Error(t, err)
	assert.Nil(t, dubboService)
	assert.Contains(t, err.Error(), "istio service cannot be nil")
}

func BenchmarkServiceConverter_ConvertToDubboService(b *testing.B) {
	converter := NewServiceConverter("test-cluster")
	istioService := &IstioServiceInfo{
		Hostname:  "test.example.com",
		Namespace: "default",
		Ports:     []IstioPortInfo{{Name: "http", Port: 8080, Protocol: "HTTP"}},
		Labels:    map[string]string{"app": "test"},
		VirtualService: &VirtualService{
			Name:      "test-vs",
			Namespace: "default",
			Spec:      map[string]interface{}{"weight": 100},
		},
		DestinationRule: &DestinationRule{
			Name:      "test-dr",
			Namespace: "default",
			Spec:      map[string]interface{}{"subset": "v1"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertToDubboService(istioService)
		require.NoError(b, err)
	}
}
