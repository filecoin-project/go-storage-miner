package modules

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/filecoin-project/lotus/lib/statestore"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-filestore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-merkledag"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/deals"
	"github.com/filecoin-project/lotus/datatransfer"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
)

func ClientFstore(r repo.LockedRepo) (dtypes.ClientFilestore, error) {
	clientds, err := r.Datastore("/client")
	if err != nil {
		return nil, err
	}
	blocks := namespace.Wrap(clientds, datastore.NewKey("blocks"))

	fm := filestore.NewFileManager(clientds, filepath.Dir(r.Path()))
	fm.AllowFiles = true
	// TODO: fm.AllowUrls (needs more code in client import)

	bs := blockstore.NewBlockstore(blocks)
	return filestore.NewFilestore(bs, fm), nil
}

func ClientBlockstore(fstore dtypes.ClientFilestore) dtypes.ClientBlockstore {
	return blockstore.NewIdStore((*filestore.Filestore)(fstore))
}

// RegisterClientValidator is an initialization hook that registers the client
// request validator with the data transfer module as the validator for
// StorageDataTransferVoucher types
func RegisterClientValidator(crv *deals.ClientRequestValidator, dtm dtypes.ClientDataTransfer) {
	if err := dtm.RegisterVoucherType(reflect.TypeOf(deals.StorageDataTransferVoucher{}), crv); err != nil {
		panic(err)
	}
}

// NewClientDAGServiceDataTransfer returns a data transfer manager that just
// uses the clients's Client DAG service for transfers
func NewClientDAGServiceDataTransfer(dag dtypes.ClientDAG) dtypes.ClientDataTransfer {
	return datatransfer.NewDAGServiceDataTransfer(dag)
}

// NewClientDealStore creates a statestore for the client to store its deals
func NewClientDealStore(ds dtypes.MetadataDS) dtypes.ClientDealStore {
	return statestore.New(namespace.Wrap(ds, datastore.NewKey("/deals/client")))
}

func ClientDAG(mctx helpers.MetricsCtx, lc fx.Lifecycle, ibs dtypes.ClientBlockstore, rt routing.Routing, h host.Host) dtypes.ClientDAG {
	bitswapNetwork := network.NewFromIpfsHost(h, rt)
	exch := bitswap.New(helpers.LifecycleCtx(mctx, lc), bitswapNetwork, ibs)

	bsvc := blockservice.New(ibs, exch)
	dag := merkledag.NewDAGService(bsvc)

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return bsvc.Close()
		},
	})

	return dag
}
