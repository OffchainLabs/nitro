// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

// Suite grouping: multiple test scenarios that share an expensive setup.
// Instead of building a node per scenario, the runner builds once and runs
// all scenarios on the same TestEnv.
//
// Usage:
//
//	func init() {
//	    RegisterSuite(SuiteEntry{
//	        Name:   "Retryables",
//	        Config: retryableConfig,
//	        Scenarios: []Scenario{
//	            {Name: "ImmediateSuccess", Run: immediateSuccessFn},
//	            {Name: "FailThenRetry", Run: failThenRetryFn},
//	            {Name: "Expiry", Run: expiryFn},
//	        },
//	    })
//	}

// Scenario is a single test case within a suite.
type Scenario struct {
	Name string
	Run  func(*TestEnv)
}

// SuiteEntry ties together a name, shared config, and multiple scenarios.
type SuiteEntry struct {
	Name      string
	Config    func() []*BuilderSpec
	Scenarios []Scenario
}

var globalSuiteRegistry []SuiteEntry

// RegisterSuite adds a suite to the global registry.
// Call from init() in each suite file.
func RegisterSuite(entry SuiteEntry) {
	globalSuiteRegistry = append(globalSuiteRegistry, entry)
}

// GetSuiteRegistry returns all registered suite entries.
func GetSuiteRegistry() []SuiteEntry {
	return globalSuiteRegistry
}
