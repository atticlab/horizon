package txsub

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/strkey"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions"
	"golang.org/x/net/context"
)

func extractEnvelopeInfo(ctx context.Context, env string, passphrase string) (result transactions.EnvelopeInfo, err error) {
	result.Tx = new(xdr.TransactionEnvelope)

	err = xdr.SafeUnmarshalBase64(env, result.Tx)

	if err != nil {
		err = &results.MalformedTransactionError{env}
		return
	}

	txb := build.TransactionBuilder{TX: &result.Tx.Tx}
	txb.Mutate(build.Network{passphrase})

	result.ContentHash, err = txb.HashHex()
	if err != nil {
		return
	}

	result.Sequence = uint64(result.Tx.Tx.SeqNum)

	aid := result.Tx.Tx.SourceAccount.MustEd25519()
	result.SourceAddress, err = strkey.Encode(strkey.VersionByteAccountID, aid[:])

	return
}
