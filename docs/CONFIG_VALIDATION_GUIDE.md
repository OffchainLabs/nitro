# Configuration Validation Guide

This guide explains how configuration validation works in Nitro and how to properly implement validation for new configuration options.

## Overview

Nitro uses a three-tier validation approach:

1. **Component-Level Validation**: Each config struct validates its own fields
2. **Cross-Component Validation**: Top-level configs validate relationships between components  
3. **Runtime Validation**: Node creation validates against external environment

## Adding Component-Level Validation

When creating a new configuration struct, implement a `Validate() error` method:

```go
type MyComponentConfig struct {
    Enable   bool          `koanf:"enable"`
    Interval time.Duration `koanf:"interval"`
    Count    int           `koanf:"count"`
}

func (c *MyComponentConfig) Validate() error {
    if c.Enable && c.Interval <= 0 {
        return fmt.Errorf("interval must be positive when component is enabled")
    }
    if c.Count < 0 {
        return fmt.Errorf("count cannot be negative")
    }
    return nil
}
```

Then call it from the parent config's validation method:

```go
func (c *ParentConfig) Validate() error {
    if err := c.MyComponent.Validate(); err != nil {
        return err
    }
    // ... other validations
    return nil
}
```

## Adding Cross-Component Validation

Cross-component validations go in the top-level config's `Validate()` method:

```go
func (c *NodeConfig) Validate() error {
    // First, validate all components
    if err := c.ComponentA.Validate(); err != nil {
        return err
    }
    if err := c.ComponentB.Validate(); err != nil {
        return err
    }
    
    // Then, validate cross-component relationships
    if c.ComponentA.Enable && c.ComponentB.Mode == "incompatible" {
        return errors.New("ComponentA cannot be enabled when ComponentB is in incompatible mode")
    }
    
    return nil
}
```

## Adding Runtime Validation

Runtime validations that require external resources go in node creation functions:

```go
func createNodeImpl(...) (*Node, error) {
    config := configFetcher.Get()
    
    // Runtime validation - check if external service is available
    if config.ExternalService.Enable {
        if err := checkExternalServiceConnectivity(config.ExternalService.URL); err != nil {
            return nil, fmt.Errorf("failed to connect to external service: %w", err)
        }
    }
    
    // ... continue with node creation
}
```

## Testing Validation

Always add tests for your validation logic:

```go
func TestMyComponentConfigValidation(t *testing.T) {
    t.Run("ValidConfig", func(t *testing.T) {
        config := MyComponentConfig{
            Enable:   true,
            Interval: time.Minute,
            Count:    10,
        }
        if err := config.Validate(); err != nil {
            t.Errorf("Expected no error, got: %v", err)
        }
    })
    
    t.Run("InvalidInterval", func(t *testing.T) {
        config := MyComponentConfig{
            Enable:   true,
            Interval: -time.Second,
            Count:    10,
        }
        if err := config.Validate(); err == nil {
            t.Error("Expected validation error for negative interval")
        }
    })
}
```

## Common Patterns

### Conditional Validation
Only validate fields when a component is enabled:

```go
func (c *MyConfig) Validate() error {
    if c.Enable {
        if c.RequiredField == "" {
            return fmt.Errorf("required-field must be set when component is enabled")
        }
    }
    return nil
}
```

### Range Validation
Validate numeric ranges:

```go
func (c *MyConfig) Validate() error {
    if c.Port < 1024 || c.Port > 65535 {
        return fmt.Errorf("port must be between 1024 and 65535, got %d", c.Port)
    }
    return nil
}
```

### URL/Path Validation
Validate URLs and file paths at the component level, but defer connectivity/existence checks to runtime validation.

## Error Messages

Make error messages clear and actionable:

- ✅ `"batch-size must be positive when batch posting is enabled"`
- ❌ `"invalid batch size"`

Include the problematic value when helpful:

- ✅ `"port must be between 1024 and 65535, got 80"`
- ❌ `"invalid port"`