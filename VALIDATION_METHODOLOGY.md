# Config Validation Methodology

This document defines a clear methodology for where config validation checks should be placed in the Nitro codebase.

## Problem Statement

Previously, config validation checks were scattered across two main areas:
1. In `cmd/nitro/nitro.go` during config parsing
2. In `arbnode/node.go` during node creation

This created confusion about where different types of validation should be placed and led to potential duplication or inconsistent validation logic.

## Validation Categories

Config validations are categorized into three types, each with a designated location:

### 1. Component-Level Validations
**Location**: Individual config struct `Validate()` methods (e.g., `BatchPosterConfig.Validate()`)

**Purpose**: Validate single component configuration consistency

**Examples**:
- Field value ranges (e.g., batch size > 0)
- Internal component consistency (e.g., if feature X is enabled, field Y must be set)
- Single component constraint validation

**Implementation**: Each config struct implements its own `Validate() error` method

### 2. Cross-Component Validations
**Location**: Top-level config `Validate()` methods (e.g., `NodeConfig.Validate()`)

**Purpose**: Validate configuration consistency across multiple components

**Examples**:
- "If validator is required, then execution.caching.state-scheme cannot be 'path'"
- "If sequencer is enabled and parent-chain-reader is enabled, then delayed-sequencer should be enabled"
- "Archive nodes must have specific cache configurations"

**Implementation**: Performed in the top-level config's `Validate()` method, after all component-level validations pass

### 3. Runtime/Environmental Validations
**Location**: Node creation functions (e.g., `createNodeImpl()` and helper functions)

**Purpose**: Validate configuration against runtime environment and external dependencies

**Examples**:
- Network connectivity checks
- File system permissions and paths
- External service availability
- Resource availability (memory, disk space)
- L1 chain connectivity and contract validation

**Implementation**: Performed during node initialization, where access to runtime context is available

## Validation Flow

```
1. Parse CLI args and config files
2. Component-Level Validations (in each config's Validate() method)
3. Cross-Component Validations (in top-level config's Validate() method)
4. [Config is now validated for parsing phase]
5. Runtime/Environmental Validations (during createNodeImpl())
6. [Node creation proceeds]
```

## Implementation Status

As of this implementation:

✅ **Component-Level Validations**: Most components already implement `Validate()` methods
- `MessagePrunerConfig.Validate()` - Added as example
- `BatchPosterConfig.Validate()` - Existing
- `MaintenanceConfig.Validate()` - Existing  
- `InboxReaderConfig.Validate()` - Existing
- And others...

✅ **Cross-Component Validations**: Implemented in `NodeConfig.Validate()`
- Validator + path scheme compatibility check - Existing
- Archive mode + message pruner warning - Added

✅ **Runtime Validations**: Properly placed in `createNodeImpl()`
- Database connectivity checks
- L1 client connectivity
- External service initialization
- Resource allocation

## Guidelines

### DO:
- Place basic field validation in component-level `Validate()` methods
- Place cross-component logic in top-level `Validate()` methods
- Place runtime checks in node creation functions
- Use descriptive error messages that help users understand what went wrong
- Document complex validation logic with comments

### DON'T:
- Mix validation types in the wrong locations
- Duplicate validation logic across multiple places
- Perform expensive runtime checks during config parsing
- Perform basic config checks during node creation

## Migration Strategy

When moving existing validations to follow this methodology:

1. Identify the validation type (component, cross-component, or runtime)
2. Move to appropriate location based on the guidelines above
3. Ensure error messages remain helpful and consistent
4. Add tests to verify the validation works correctly
5. Remove any duplicate validations

## Benefits

This methodology provides:
- **Clarity**: Developers know exactly where to place validation logic
- **Performance**: Expensive runtime checks happen only when necessary
- **Maintainability**: Validations are logically grouped and easy to find
- **Testability**: Each validation type can be tested independently
- **User Experience**: Users get validation feedback at the appropriate time

## Examples in Code

See:
- `arbnode/message_pruner.go` - Component-level validation example
- `cmd/nitro/nitro.go` - Cross-component validation examples
- `arbnode/node.go` - Runtime validation examples during node creation
- Tests in `arbnode/message_pruner_test.go` and `cmd/nitro/config_test.go`