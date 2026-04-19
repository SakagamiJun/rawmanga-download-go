package settings

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
	"github.com/sakagamijun/rawmanga-download-go/internal/store"
)

type Service struct {
	store *store.SQLiteStore
	cache contracts.AppSettings
}

func NewService(store *store.SQLiteStore) (*Service, error) {
	service := &Service{
		store: store,
		cache: DefaultSettings(),
	}

	settings, found, err := store.GetSettings()
	if err != nil {
		return nil, err
	}

	if !found {
		if err := store.SaveSettings(service.cache); err != nil {
			return nil, err
		}

		return service, nil
	}

	normalized, err := service.Normalize(settings)
	if err != nil {
		return nil, err
	}

	service.cache = normalized
	if err := store.SaveSettings(normalized); err != nil {
		return nil, err
	}

	return service, nil
}

func DefaultSettings() contracts.AppSettings {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return contracts.AppSettings{
		OutputRoot:             filepath.Join(homeDir, "Downloads", "KLZ9"),
		MaxConcurrentDownloads: 6,
		RetryCount:             3,
		RequestTimeoutSec:      30,
		LocaleMode:             contracts.LocaleModeSystem,
		Locale:                 "en",
		ThemeMode:              contracts.ThemeModeSystem,
	}
}

func (s *Service) Get() contracts.AppSettings {
	return s.cache
}

func (s *Service) Update(input contracts.AppSettings) (contracts.AppSettings, error) {
	normalized, err := s.Normalize(input)
	if err != nil {
		return contracts.AppSettings{}, err
	}

	if err := s.store.SaveSettings(normalized); err != nil {
		return contracts.AppSettings{}, err
	}

	s.cache = normalized
	return normalized, nil
}

func (s *Service) Normalize(input contracts.AppSettings) (contracts.AppSettings, error) {
	settings := DefaultSettings()

	if input.OutputRoot != "" {
		settings.OutputRoot = input.OutputRoot
	}

	if input.MaxConcurrentDownloads > 0 {
		settings.MaxConcurrentDownloads = input.MaxConcurrentDownloads
	}

	if input.RetryCount >= 0 {
		settings.RetryCount = input.RetryCount
	}

	if input.RequestTimeoutSec > 0 {
		settings.RequestTimeoutSec = input.RequestTimeoutSec
	}

	switch input.LocaleMode {
	case "", contracts.LocaleModeSystem:
		settings.LocaleMode = contracts.LocaleModeSystem
	case contracts.LocaleModeManual:
		settings.LocaleMode = contracts.LocaleModeManual
	default:
		return contracts.AppSettings{}, contracts.ContractError{
			Code:    contracts.ErrCodeSettingsInvalid,
			Message: fmt.Sprintf("unsupported locale mode: %s", input.LocaleMode),
		}
	}

	switch input.Locale {
	case "", "en":
		settings.Locale = "en"
	case "zh-CN", "ja":
		settings.Locale = input.Locale
	default:
		return contracts.AppSettings{}, contracts.ContractError{
			Code:    contracts.ErrCodeSettingsInvalid,
			Message: fmt.Sprintf("unsupported locale: %s", input.Locale),
		}
	}

	switch input.ThemeMode {
	case "", contracts.ThemeModeSystem:
		settings.ThemeMode = contracts.ThemeModeSystem
	case contracts.ThemeModeLight, contracts.ThemeModeDark:
		settings.ThemeMode = input.ThemeMode
	default:
		return contracts.AppSettings{}, contracts.ContractError{
			Code:    contracts.ErrCodeSettingsInvalid,
			Message: fmt.Sprintf("unsupported theme mode: %s", input.ThemeMode),
		}
	}

	return settings, nil
}
