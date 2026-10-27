package utils

import "errors"

type UICustomizationPolicyConfig struct {
	AllowWorldCSS       bool `json:"allowWorldCss" yaml:"allowWorldCss"`
	AllowWorldJS        bool `json:"allowWorldJs" yaml:"allowWorldJs"`
	AllowPublicWorldCSS bool `json:"allowPublicWorldCss" yaml:"allowPublicWorldCss"`
}

func NormalizeUICustomizationPolicy(cfg *UICustomizationPolicyConfig) *UICustomizationPolicyConfig {
	if cfg == nil {
		// World scripts are opt-in: a missing allowWorldJs key stays false.
		return &UICustomizationPolicyConfig{AllowWorldCSS: true, AllowPublicWorldCSS: true}
	}
	copy := *cfg
	return &copy
}

func ValidateUICustomizationPolicy(cfg *UICustomizationPolicyConfig) error {
	if cfg == nil {
		return errors.New("UI customization policy must be normalized")
	}
	// All switches are booleans: false is an explicit policy, never a default.
	return nil
}
