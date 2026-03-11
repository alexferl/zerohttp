package zerohttp

import "github.com/alexferl/zerohttp/config"

// mergeRecoverConfig merges user config with defaults
func mergeRecoverConfig(defaultCfg, userCfg config.RecoverConfig) config.RecoverConfig {
	if userCfg.StackSize != 0 {
		defaultCfg.StackSize = userCfg.StackSize
	}
	if userCfg.EnableStackTrace {
		defaultCfg.EnableStackTrace = userCfg.EnableStackTrace
	}
	return defaultCfg
}

// mergeRequestBodySizeConfig merges user config with defaults
func mergeRequestBodySizeConfig(defaultCfg, userCfg config.RequestBodySizeConfig) config.RequestBodySizeConfig {
	if userCfg.MaxBytes != 0 {
		defaultCfg.MaxBytes = userCfg.MaxBytes
	}
	if len(userCfg.ExemptPaths) > 0 {
		defaultCfg.ExemptPaths = userCfg.ExemptPaths
	}
	return defaultCfg
}

// mergeRequestIDConfig merges user config with defaults
func mergeRequestIDConfig(defaultCfg, userCfg config.RequestIDConfig) config.RequestIDConfig {
	if userCfg.Header != "" {
		defaultCfg.Header = userCfg.Header
	}
	if userCfg.Generator != nil {
		defaultCfg.Generator = userCfg.Generator
	}
	if userCfg.ContextKey != "" {
		defaultCfg.ContextKey = userCfg.ContextKey
	}
	return defaultCfg
}

// mergeRequestLoggerConfig merges user config with defaults
func mergeRequestLoggerConfig(defaultCfg, userCfg config.RequestLoggerConfig) config.RequestLoggerConfig {
	if userCfg.LogErrors {
		defaultCfg.LogErrors = userCfg.LogErrors
	}
	if len(userCfg.Fields) > 0 {
		defaultCfg.Fields = userCfg.Fields
	}
	if len(userCfg.ExemptPaths) > 0 {
		defaultCfg.ExemptPaths = userCfg.ExemptPaths
	}
	return defaultCfg
}

// mergeSecurityHeadersConfig merges user config with defaults
func mergeSecurityHeadersConfig(defaultCfg, userCfg config.SecurityHeadersConfig) config.SecurityHeadersConfig {
	if userCfg.ContentSecurityPolicy != "" {
		defaultCfg.ContentSecurityPolicy = userCfg.ContentSecurityPolicy
	}
	defaultCfg.ContentSecurityPolicyReportOnly = userCfg.ContentSecurityPolicyReportOnly
	if userCfg.CrossOriginEmbedderPolicy != "" {
		defaultCfg.CrossOriginEmbedderPolicy = userCfg.CrossOriginEmbedderPolicy
	}
	if userCfg.CrossOriginOpenerPolicy != "" {
		defaultCfg.CrossOriginOpenerPolicy = userCfg.CrossOriginOpenerPolicy
	}
	if userCfg.CrossOriginResourcePolicy != "" {
		defaultCfg.CrossOriginResourcePolicy = userCfg.CrossOriginResourcePolicy
	}
	if userCfg.PermissionsPolicy != "" {
		defaultCfg.PermissionsPolicy = userCfg.PermissionsPolicy
	}
	if userCfg.ReferrerPolicy != "" {
		defaultCfg.ReferrerPolicy = userCfg.ReferrerPolicy
	}
	if userCfg.Server != "" {
		defaultCfg.Server = userCfg.Server
	}
	if userCfg.StrictTransportSecurity.MaxAge != 0 {
		defaultCfg.StrictTransportSecurity = userCfg.StrictTransportSecurity
	}
	if userCfg.XContentTypeOptions != "" {
		defaultCfg.XContentTypeOptions = userCfg.XContentTypeOptions
	}
	if userCfg.XFrameOptions != "" {
		defaultCfg.XFrameOptions = userCfg.XFrameOptions
	}
	if len(userCfg.ExemptPaths) > 0 {
		defaultCfg.ExemptPaths = userCfg.ExemptPaths
	}
	return defaultCfg
}

// mergeMetricsConfig merges user config with defaults
func mergeMetricsConfig(defaultCfg, userCfg config.MetricsConfig) config.MetricsConfig {
	defaultCfg.Enabled = userCfg.Enabled
	if userCfg.Endpoint != "" {
		defaultCfg.Endpoint = userCfg.Endpoint
	}
	// ServerAddr can be explicitly set to empty string to disable separate metrics server
	defaultCfg.ServerAddr = userCfg.ServerAddr
	if len(userCfg.DurationBuckets) > 0 {
		defaultCfg.DurationBuckets = userCfg.DurationBuckets
	}
	if len(userCfg.SizeBuckets) > 0 {
		defaultCfg.SizeBuckets = userCfg.SizeBuckets
	}
	if len(userCfg.ExcludePaths) > 0 {
		defaultCfg.ExcludePaths = userCfg.ExcludePaths
	}
	if userCfg.PathLabelFunc != nil {
		defaultCfg.PathLabelFunc = userCfg.PathLabelFunc
	}
	if userCfg.CustomLabels != nil {
		defaultCfg.CustomLabels = userCfg.CustomLabels
	}
	return defaultCfg
}
