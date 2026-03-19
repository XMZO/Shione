package frpembed

import (
	"context"
	"fmt"
	stdlog "log"
	"path/filepath"
	"time"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/featuregate"
	"github.com/fatedier/frp/pkg/policy/security"
	frplog "github.com/fatedier/frp/pkg/util/log"
)

// RunnableService is the minimal frp service surface the host needs.
type RunnableService interface {
	Run(ctx context.Context) error
	GracefulClose(d time.Duration)
}

// Builder validates and constructs embeddable frp services.
type Builder interface {
	Validate(configPath string) error
	Build(configPath string) (RunnableService, string, error)
}

// LocalBuilder uses the vendored frp module directly.
type LocalBuilder struct {
	StrictConfig  bool
	AllowedUnsafe []string
}

func NewLocalBuilder() *LocalBuilder {
	return &LocalBuilder{
		StrictConfig: true,
	}
}

func (b *LocalBuilder) Validate(configPath string) error {
	_, _, err := b.prepareService(configPath)
	return err
}

func (b *LocalBuilder) Build(configPath string) (RunnableService, string, error) {
	return b.prepareService(configPath)
}

func (b *LocalBuilder) prepareService(configPath string) (RunnableService, string, error) {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("resolve frp config path: %w", err)
	}

	result, err := config.LoadClientConfigResult(absConfigPath, b.StrictConfig)
	if err != nil {
		return nil, "", err
	}
	if result.IsLegacyFormat {
		stdlog.Printf("frp warning: legacy ini config detected for %s", absConfigPath)
	}
	if len(result.Common.FeatureGates) > 0 {
		if err := featuregate.SetFromMap(result.Common.FeatureGates); err != nil {
			return nil, "", err
		}
	}

	aggregator, err := buildAggregator(result, absConfigPath)
	if err != nil {
		return nil, "", err
	}

	unsafeFeatures := security.NewUnsafeFeatures(b.AllowedUnsafe)
	proxyCfgs, visitorCfgs, err := aggregator.Load()
	if err != nil {
		return nil, "", fmt.Errorf("load frp config from sources: %w", err)
	}
	proxyCfgs, visitorCfgs = config.FilterClientConfigurers(result.Common, proxyCfgs, visitorCfgs)
	proxyCfgs = config.CompleteProxyConfigurers(proxyCfgs)
	visitorCfgs = config.CompleteVisitorConfigurers(visitorCfgs)

	warning, err := validation.ValidateAllClientConfig(result.Common, proxyCfgs, visitorCfgs, unsafeFeatures)
	if warning != nil {
		stdlog.Printf("frp config warning: %v", warning)
	}
	if err != nil {
		return nil, "", err
	}

	frplog.InitLogger(
		result.Common.Log.To,
		result.Common.Log.Level,
		int(result.Common.Log.MaxDays),
		result.Common.Log.DisablePrintColor,
	)

	svc, err := client.NewService(client.ServiceOptions{
		Common:                 result.Common,
		ConfigSourceAggregator: aggregator,
		UnsafeFeatures:         unsafeFeatures,
		ConfigFilePath:         absConfigPath,
	})
	if err != nil {
		return nil, "", err
	}

	return svc, absConfigPath, nil
}

func buildAggregator(result *config.ClientConfigLoadResult, configPath string) (*source.Aggregator, error) {
	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(result.Proxies, result.Visitors); err != nil {
		return nil, fmt.Errorf("set frp config source: %w", err)
	}

	var storeSource *source.StoreSource
	if result.Common.Store.IsEnabled() {
		storePath := result.Common.Store.Path
		if storePath != "" && !filepath.IsAbs(storePath) {
			storePath = filepath.Join(filepath.Dir(configPath), storePath)
		}

		s, err := source.NewStoreSource(source.StoreSourceConfig{
			Path: storePath,
		})
		if err != nil {
			return nil, fmt.Errorf("create frp store source: %w", err)
		}
		storeSource = s
	}

	aggregator := source.NewAggregator(configSource)
	if storeSource != nil {
		aggregator.SetStoreSource(storeSource)
	}
	return aggregator, nil
}
