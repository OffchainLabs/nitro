package ipfshelper

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const defaultFetchTimeout = 1 * time.Minute // TODO(magic)

type IpfsHelper struct {
	api      icore.CoreAPI
	node     *core.IpfsNode
	cfg      *config.Config
	repoPath string
	repo     repo.Repo
}

func (h *IpfsHelper) createRepo(repoDirectory string, profiles string) error {
	fileInfo, err := os.Stat(repoDirectory)
	if err != nil {
		return fmt.Errorf("failed to stat ipfs repo directory, %s : %w", repoDirectory, err)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", repoDirectory)
	}
	h.repoPath = repoDirectory
	// Create a config with default options and a 2048 bit key
	h.cfg, err = config.Init(io.Discard, 2048)
	if err != nil {
		return err
	}
	if len(profiles) > 0 {
		for _, profile := range strings.Split(profiles, ",") {
			transformer, ok := config.Profiles[profile]
			if !ok {
				return fmt.Errorf("invalid ipfs configuration profile: %s", profile)
			}

			if err := transformer.Transform(h.cfg); err != nil {
				return err
			}
		}
	}
	// Create the repo with the config
	err = fsrepo.Init(h.repoPath, h.cfg)
	if err != nil {
		return fmt.Errorf("failed to init ipfs repo: %w", err)
	}
	h.repo, err = fsrepo.Open(h.repoPath)
	if err != nil {
		return fmt.Errorf("failed to open ipfs repo: %w", err)
	}
	return nil
}

func (h *IpfsHelper) createNode(ctx context.Context, clientOnly bool) error {
	var routing libp2p.RoutingOption
	if clientOnly {
		routing = libp2p.DHTClientOption
	} else {
		routing = libp2p.DHTOption
	}
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: routing,
		Repo:    h.repo,
	}
	var err error
	h.node, err = core.NewNode(ctx, nodeOptions)
	if err != nil {
		return err
	}
	h.api, err = coreapi.NewCoreAPI(h.node)
	return err
}

func (h *IpfsHelper) connectToPeers(ctx context.Context, peers []string) error {
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addressString := range peers {
		address, err := ma.NewMultiaddr(addressString)
		if err != nil {
			return err
		}
		addressInfo, err := peer.AddrInfoFromP2pAddr(address)
		if err != nil {
			return err
		}
		peerInfo, ok := peerInfos[addressInfo.ID]
		if !ok {
			peerInfo = &peer.AddrInfo{ID: addressInfo.ID}
			peerInfos[peerInfo.ID] = peerInfo
		}
		peerInfo.Addrs = append(peerInfo.Addrs, addressInfo.Addrs...)
	}
	var wg sync.WaitGroup
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			err := h.api.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Warn("failed to connect to peer", "peerId", peerInfo.ID, "err", err)
				return
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func (h *IpfsHelper) GetPeerHostAddresses() ([]string, error) {
	addresses, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(h.node.PeerHost))
	if err != nil {
		return []string{}, err
	}
	addressesStrings := make([]string, len(addresses))
	for i, a := range addresses {
		addressesStrings[i] = a.String()
	}
	return addressesStrings, nil
}

func (h *IpfsHelper) DownloadFile(ctx context.Context, cidString string, destinationDirectory string) (string, error) {
	return h.downloadFileImpl(ctx, cidString, destinationDirectory, defaultFetchTimeout)
}

func (h *IpfsHelper) downloadFileImpl(ctx context.Context, cidString string, destinationDirectory string, fetchTimeout time.Duration) (string, error) {
	cidPath := icorepath.New(cidString)
	var fetchCtx context.Context
	if fetchTimeout > 0 {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, fetchTimeout)
		defer cancel()
		fetchCtx = ctxWithTimeout
	} else {
		fetchCtx = ctx
	}
	resolvedPath, err := h.api.ResolvePath(fetchCtx, cidPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	rootNodeDirectory, err := h.api.Unixfs().Get(fetchCtx, cidPath)
	if err != nil {
		return "", fmt.Errorf("could not get file with CID: %w", err)
	}
	outputFilePath := filepath.Join(destinationDirectory, resolvedPath.Cid().String())
	err = files.WriteTo(rootNodeDirectory, outputFilePath)
	if err != nil {
		return "", fmt.Errorf("could not write out the fetched CID: %w", err)
	}
	return outputFilePath, nil
}

func (h *IpfsHelper) AddFile(ctx context.Context, filePath string, includeHidden bool) (path.Resolved, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	fileNode, err := files.NewSerialFile(filePath, includeHidden, fileInfo)
	if err != nil {
		return nil, err
	}
	return h.api.Unixfs().Add(ctx, fileNode)
}

func CreateIpfsHelper(ctx context.Context, repoDirectory string, clientOnly bool) (*IpfsHelper, error) {
	return createIpfsHelperImpl(ctx, repoDirectory, clientOnly, []string{}, "")
}

func (h *IpfsHelper) Close() error {
	return h.node.Close()
}

func setupPlugins() error {
	plugins, err := loader.NewPluginLoader("")
	if err != nil {
		return fmt.Errorf("error loading plugins: %w", err)
	}
	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %w", err)
	}
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %w", err)
	}
	return nil
}

var loadPluginsOnce sync.Once

func createIpfsHelperImpl(ctx context.Context, repoDirectory string, clientOnly bool, peerList []string, profiles string) (*IpfsHelper, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins()
	})
	if onceErr != nil {
		return nil, onceErr
	}
	client := IpfsHelper{}
	err := client.createRepo(repoDirectory, profiles)
	if err != nil {
		return nil, err
	}
	err = client.createNode(ctx, clientOnly)
	if err != nil {
		return nil, err
	}
	err = client.connectToPeers(ctx, peerList)
	if err != nil {
		return nil, err
	}
	return &client, nil
}
