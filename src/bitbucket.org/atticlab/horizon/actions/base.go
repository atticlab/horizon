package actions

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"bitbucket.org/atticlab/go-smart-base/hash"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/render/sse"
	gctx "github.com/goji/context"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
)

// Base is a helper struct you can use as part of a custom action via
// composition.
//
// TODO: example usage
type Base struct {
	Ctx     context.Context
	GojiCtx web.C
	W       http.ResponseWriter
	R       *http.Request
	Err     error

	isSetup  bool
	IsSigned bool
	Signer   string
}

// Prepare established the common attributes that get used in nearly every
// action.  "Child" actions may override this method to extend action, but it
// is advised you also call this implementation to maintain behavior.
func (base *Base) Prepare(c web.C, w http.ResponseWriter, r *http.Request) {
	base.Ctx = gctx.FromC(c)
	base.GojiCtx = c
	base.W = w
	base.R = r
	base.IsSigned = false
	signature := r.Header.Get("X-AuthSignature")
	publicKey := r.Header.Get("X-AuthPublicKey")
	timestamp := r.Header.Get("X-AuthTimestamp")
	if signature == "" || publicKey == "" || timestamp == "" {
		return
	}
	base.Signer = publicKey
	// Read the content
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(r.Body)
	}
	// Restore the io.ReadCloser to its original state
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	// Use the content
	bodyString := string(bodyBytes)

	signatureBase := "{method: 'post', body: '" + bodyString + "', timestamp: '" + timestamp + "'}"
	hashBase := hash.Hash([]byte(signatureBase))
	actual := hex.EncodeToString(hashBase[:])
	log.WithField("signatureBase", signatureBase).
		Info("signatureBase")
	log.WithField("actual", actual).
		WithField("hashBase", hashBase).
		WithField("publicKey", publicKey).
		Info("signatureBase")
	pubKey, err := keypair.Parse(publicKey)
	if err != nil {
		log.WithField("err", err).
			Info("signatureBase")
		return
	}
	var decoratedSign xdr.DecoratedSignature
	err = xdr.SafeUnmarshalBase64(signature, &decoratedSign)
	if err != nil {
		log.WithField("err", err).
			Info("signatureBase")
		return
	}

	log.WithField("sigBytes", decoratedSign.Signature).
		WithField("signature", signature).
		Info("signatureBase")
	err = pubKey.Verify(hashBase[:], decoratedSign.Signature)
	if err == nil {
		log.Info("signatureVerified")
		base.IsSigned = true
	} else {
		log.Info("signatureNotVerified")
		log.WithField("err", err).
			Error("signatureBase")
	}
	// let data = hash(signatureBase);

}

// Execute trigger content negottion and the actual execution of one of the
// action's handlers.
func (base *Base) Execute(action interface{}) {
	contentType := render.Negotiate(base.Ctx, base.R)

	switch contentType {
	case render.MimeHal, render.MimeJSON:
		action, ok := action.(JSON)

		if !ok {
			goto NotAcceptable
		}

		action.JSON()

		if base.Err != nil {
			problem.Render(base.Ctx, base.W, base.Err)
			return
		}

	case render.MimeEventStream:
		action, ok := action.(SSE)
		if !ok {
			goto NotAcceptable
		}

		stream, ok := sse.NewStream(base.Ctx, base.W, base.R)
		if !ok {
			return
		}

		for {
			action.SSE(stream)

			if base.Err != nil {
				stream.Err(base.Err)
			}

			if stream.IsDone() {
				return
			}

			select {
			case <-base.Ctx.Done():
				return
			case <-sse.Pumped():
				//no-op, continue onto the next iteration
			}

		}
	case render.MimeRaw:
		action, ok := action.(Raw)

		if !ok {
			goto NotAcceptable
		}

		action.Raw()

		if base.Err != nil {
			problem.Render(base.Ctx, base.W, base.Err)
			return
		}
	default:
		goto NotAcceptable
	}
	return

NotAcceptable:
	problem.Render(base.Ctx, base.W, problem.NotAcceptable)
	return
}

// Do executes the provided func iff there is no current error for the action.
// Provides a nicer way to invoke a set of steps that each may set `action.Err`
// during execution
func (base *Base) Do(fns ...func()) {
	for _, fn := range fns {
		if base.Err != nil {
			return
		}

		fn()
	}
}

// Setup runs the provided funcs if and only if no call to Setup() has been
// made previously on this action.
func (base *Base) Setup(fns ...func()) {
	if base.isSetup {
		return
	}
	base.Do(fns...)
	base.isSetup = true
}
