package blockchain

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/pkg/errors"
)

const (
	// BucketName is where we store the nfts
	BucketName = "blknft"
)

type BlockchainNFT interface {
	nft.BaseNFT
	UpdateDetails(actor weave.Address, details TokenDetails) error
	Details() TokenDetails
}
type BlockchainNFTAdapter struct {
	*nft.NonFungibleToken
}

func (a *BlockchainNFTAdapter) Details() TokenDetails {
	return TokenDetails{
		// todo: map all
		ChainID: a.GetDetails().GetBlockchain().ChainID,
	}
}
func (a *BlockchainNFTAdapter) UpdateDetails(actor weave.Address, details TokenDetails) error {
	newDetails := &nft.TokenDetails_Blockchain{
		Blockchain: &nft.BlockChainDetails{
			// todo: map all
			ChainID: details.ChainID,
		},
	}
	return a.TakeAction(actor, nft.ActionKind_updatePayloadApproval, newDetails)
}

// As BlockchainNFT will safely type-cast any value from Bucket
func AsBlockchainNFT(obj orm.Object) (BlockchainNFT, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*nft.NonFungibleToken)
	if !ok {
		return nil, errors.New("unsupported type") // todo: move
	}
	return &BlockchainNFTAdapter{x}, nil
}

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
}

//var _ nft.BaseBucket = Bucket{}

func NewBucket() Bucket {
	return Bucket{
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, nft.NewNonFungibleToken(nil, nil))),
	}
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, key []byte, details TokenDetails) (orm.Object, error) {
	obj, err := b.Get(db, key)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.New("key exists already") // todo: move into errors file
	}
	obj = nft.NewNonFungibleToken(key, owner)
	//blockchain, err := AsBlockchainNFT(obj)
	//if err != nil {
	//	return nil, err
	//}
	//blockchain.SetPubKey(owner, pubKey)
	return obj, nil
}
