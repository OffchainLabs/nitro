package ipfshelper

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type IpfsHelper struct {
	api      icore.CoreAPI
	node     *core.IpfsNode
	cfg      *config.Config
	repoPath string
	repo     repo.Repo
}

func (h *IpfsHelper) createRepo(repoDirectory string) error {
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
	// TODO(magic) refactor?
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

func (h *IpfsHelper) DownloadFile(ctx context.Context, cidString string, destinationDirectory string) (string, error) {
	cidPath := icorepath.New(cidString)
	rootNodeDirectory, err := h.api.Unixfs().Get(ctx, cidPath)
	if err != nil {
		return "", fmt.Errorf("could not get file with CID: %w", err)
	}
	// TODO(magic) fix creating output file path to support cidString with protocol prefix e.g. "/ipfs/..."
	outputFilePath := filepath.Join(destinationDirectory, cidString)
	err = files.WriteTo(rootNodeDirectory, outputFilePath)
	if err != nil {
		return "", fmt.Errorf("could not write out the fetched CID: %w", err)
	}
	return outputFilePath, nil
}

func CreateIpfsHelper(ctx context.Context, repoDirectory string, clientOnly bool) (*IpfsHelper, error) {
	return createIpfsHelperImpl(ctx, repoDirectory, clientOnly, []string{})
}

func (h *IpfsHelper) Close() error {
	return h.node.Close()
}

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
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

func createIpfsHelperImpl(ctx context.Context, repoDirectory string, clientOnly bool, peerList []string) (*IpfsHelper, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, onceErr
	}
	client := IpfsHelper{}
	err := client.createRepo(repoDirectory)
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
