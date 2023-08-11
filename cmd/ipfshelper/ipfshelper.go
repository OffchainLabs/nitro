package ipfshelper

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	iface "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/coreiface/options"
	"github.com/ipfs/go-libipfs/files"
	"github.com/ipfs/interface-go-ipfs-core/path"
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

const DefaultIpfsProfiles = ""

type IpfsHelper struct {
	api      iface.CoreAPI
	node     *core.IpfsNode
	cfg      *config.Config
	repoPath string
	repo     repo.Repo
}

func (h *IpfsHelper) createRepo(downloadPath string, profiles string) error {
	fileInfo, err := os.Stat(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to stat ipfs repo directory: %w", err)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", downloadPath)
	}
	h.repoPath = filepath.Join(downloadPath, "ipfs-repo")
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
	// fsrepo.Init initializes new repo only if it's not initialized yet
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

func normalizeCidString(cidString string) string {
	if strings.HasPrefix(cidString, "ipfs://") {
		return "/ipfs/" + cidString[7:]
	}
	if strings.HasPrefix(cidString, "ipns://") {
		return "/ipns/" + cidString[7:]
	}
	return cidString
}

func (h *IpfsHelper) DownloadFile(ctx context.Context, cidString string, destinationDir string) (string, error) {
	cidString = normalizeCidString(cidString)
	cidPath := path.New(cidString)
	resolvedPath, err := h.api.ResolvePath(ctx, cidPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	// first pin the root node, then all its children nodes in random order to improve sharing with peers started at the same time
	if err := h.api.Pin().Add(ctx, resolvedPath, options.Pin.Recursive(false)); err != nil {
		return "", fmt.Errorf("failed to pin root path: %w", err)
	}
	links, err := h.api.Object().Links(ctx, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to get root links: %w", err)
	}
	log.Info("Pinning ipfs subtrees...")
	printProgress := func(done int, all int) {
		if all == 0 {
			all = 1 // avoid division by 0
			done = 1
		}
		fmt.Printf("\033[2K\rPinned %d / %d subtrees (%.2f%%)", done, all, float32(done)/float32(all)*100)
	}
	permutation := rand.Perm(len(links))
	printProgress(0, len(links))
	for i, j := range permutation {
		link := links[j]
		if err := h.api.Pin().Add(ctx, path.IpfsPath(link.Cid), options.Pin.Recursive(true)); err != nil {
			return "", fmt.Errorf("failed to pin child path: %w", err)
		}
		printProgress(i+1, len(links))
	}
	fmt.Printf("\n")
	rootNodeDirectory, err := h.api.Unixfs().Get(ctx, cidPath)
	if err != nil {
		return "", fmt.Errorf("could not get file with CID: %w", err)
	}
	log.Info("Writing file...")
	outputFilePath := filepath.Join(destinationDir, resolvedPath.Cid().String())
	_ = os.Remove(outputFilePath)
	err = files.WriteTo(rootNodeDirectory, outputFilePath)
	if err != nil {
		return "", fmt.Errorf("could not write out the fetched CID: %w", err)
	}
	log.Info("Download done.")
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

func CreateIpfsHelper(ctx context.Context, downloadPath string, clientOnly bool, peerList []string, profiles string) (*IpfsHelper, error) {
	return createIpfsHelperImpl(ctx, downloadPath, clientOnly, peerList, profiles)
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

func createIpfsHelperImpl(ctx context.Context, downloadPath string, clientOnly bool, peerList []string, profiles string) (*IpfsHelper, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins()
	})
	if onceErr != nil {
		return nil, onceErr
	}
	client := IpfsHelper{}
	err := client.createRepo(downloadPath, profiles)
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

func CanBeIpfsPath(pathString string) bool {
	path := path.New(pathString)
	return path.IsValid() == nil ||
		strings.HasPrefix(pathString, "/ipfs/") ||
		strings.HasPrefix(pathString, "/ipld/") ||
		strings.HasPrefix(pathString, "/ipns/") ||
		strings.HasPrefix(pathString, "ipfs://") ||
		strings.HasPrefix(pathString, "ipns://")
}

// TODO break abstraction for now til we figure out what fns are needed
func (h *IpfsHelper) GetAPI() iface.CoreAPI {
	return h.api
}
