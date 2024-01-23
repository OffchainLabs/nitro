// Package server defines the client-facing API methods for fetching data
// related to BOLD challenges. It handles HTTP methods with their requests and responses.
package server

import "github.com/OffchainLabs/bold/api/backend"

type Server struct {
	backend backend.BusinessLogicProvider
}
