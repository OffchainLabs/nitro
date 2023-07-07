package staker

import l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"

var _ l2stateprovider.Provider = (*StateManager)(nil)
