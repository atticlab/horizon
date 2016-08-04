package transactions

import "bitbucket.org/atticlab/go-smart-base/xdr"

type EnvelopeInfo struct {
	ContentHash   string
	Sequence      uint64
	SourceAddress string
	Tx            *xdr.TransactionEnvelope
}
