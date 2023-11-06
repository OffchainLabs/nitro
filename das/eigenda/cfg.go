package eigenda

type DAConfig struct {
	Enable                    bool   `koanf:"enable"`
	DisperserRpc              string `koanf:"disperser-rpc"`
	RetrieverRpc              string `koanf:"retriever-rpc"`
	PrimaryQuorumID           uint32 `koanf:"primary-quorum-id"`
	PrimaryAdversaryThreshold uint32 `koanf:"primary-adversary-threshold"`
	PrimaryQuorumThreshold    uint32 `koanf:"primary-quorum-threshold"`
	StatusQueryRetryInterval  string `koanf:"status-query-retry-interval"`
	StatusQueryTimeout        string `koanf:"status-query-timeout"`
}
