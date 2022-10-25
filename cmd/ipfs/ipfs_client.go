package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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

type IpfsClient struct {
	// TODO(magic) remove unneeded fields
	repoPath string
	api      icore.CoreAPI
	node     *core.IpfsNode
	cfg      *config.Config
	repo     repo.Repo
}

func (c *IpfsClient) createIpfsRepo(repoDirectory string) error {
	fileInfo, err := os.Stat(repoDirectory)
	if err != nil {
		return fmt.Errorf("failed to stat ipfs repo directory, %s : %w", repoDirectory, err)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", repoDirectory)
	}
	c.repoPath = repoDirectory
	// Create a config with default options and a 2048 bit key
	c.cfg, err = config.Init(io.Discard, 2048)
	if err != nil {
		return err
	}
	// Create the repo with the config
	err = fsrepo.Init(c.repoPath, c.cfg)
	if err != nil {
		return fmt.Errorf("failed to init ipfs repo: %w", err)
	}
	c.repo, err = fsrepo.Open(c.repoPath)
	if err != nil {
		return fmt.Errorf("failed to open ipfs repo: %w", err)
	}
	return nil
}

func (c *IpfsClient) createClientNode(ctx context.Context) error {
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTClientOption,
		Repo:    c.repo,
	}
	var err error
	c.node, err = core.NewNode(ctx, nodeOptions)
	if err != nil {
		return err
	}
	c.api, err = coreapi.NewCoreAPI(c.node)
	return err
}

func (c *IpfsClient) connectToPeers(ctx context.Context, peers []string) error {
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
	var connected uint32
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			err := c.api.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Warn("failed to connect to peer", "peerId", peerInfo.ID, "err", err)
				return
			}
			atomic.AddUint32(&connected, 1)
		}(peerInfo)
	}
	wg.Wait()
	if connected == 0 {
		return errors.New("failed to connect to any peer")
	}
	return nil
}

func (c *IpfsClient) DownloadFile(ctx context.Context, cidString string, destinationDirectory string) (string, error) {
	cidPath := icorepath.New(cidString)
	rootNodeDirectory, err := c.api.Unixfs().Get(ctx, cidPath)
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

func CreateIpfsClient(ctx context.Context, repoDirectory string) (*IpfsClient, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, onceErr
	}
	client := IpfsClient{}
	err := client.createIpfsRepo(repoDirectory)
	if err != nil {
		return nil, err
	}
	err = client.createClientNode(ctx)
	if err != nil {
		return nil, err
	}
	// TODO(magic) make the peers list configurable or add more default entries
	err = client.connectToPeers(ctx, []string{"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN"})
	if err != nil {
		return nil, err
	}
	return &client, nil
}
