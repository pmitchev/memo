package memo

import (
	"bytes"
	"fmt"
	"git.jasonc.me/main/bitcoin/bitcoin/memo"
	"git.jasonc.me/main/memo/app/auth"
	"git.jasonc.me/main/memo/app/bitcoin/node"
	"git.jasonc.me/main/memo/app/bitcoin/transaction"
	"git.jasonc.me/main/memo/app/db"
	"git.jasonc.me/main/memo/app/res"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/web"
	"net/http"
)

var newRoute = web.Route{
	Pattern:    res.UrlMemoNew,
	NeedsLogin: true,
	Handler: func(r *web.Response) {
		r.Render()
	},
}

var newSubmitRoute = web.Route{
	Pattern:     res.UrlMemoNewSubmit,
	NeedsLogin:  true,
	CsrfProtect: true,
	Handler: func(r *web.Response) {
		message := r.Request.GetFormValue("message")
		password := r.Request.GetFormValue("password")
		user, err := auth.GetSessionUser(r.Session.CookieId)
		if err != nil {
			r.Error(jerr.Get("error getting session user", err), http.StatusInternalServerError)
			return
		}
		key, err := db.GetKeyForUser(user.Id)
		if err != nil {
			r.Error(jerr.Get("error getting key for user", err), http.StatusInternalServerError)
			return
		}
		transactions, err := db.GetTransactionsForPkHash(key.PkHash)
		var txOut *db.TransactionOut
		for _, txn := range transactions {
			for _, out := range txn.TxOut {
				if out.TxnIn == nil && out.Value > 1000 && bytes.Equal(out.KeyPkHash, key.PkHash) {
					txOut = out
				}
			}
		}
		if txOut == nil {
			r.Error(jerr.New("unable to find an output to spend"), http.StatusUnprocessableEntity)
			return
		}

		privateKey, err := key.GetPrivateKey(password)
		if err != nil {
			r.Error(jerr.Get("error getting private key", err), http.StatusUnauthorized)
			return
		}

		address := key.GetAddress()
		var fee = int64(283 - memo.MaxPostSize + len([]byte(message)))
		tx, err := transaction.Create(txOut, privateKey, []transaction.SpendOutput{{
			Type:    transaction.SpendOutputTypeP2PK,
			Address: address,
			Amount:  txOut.Value - fee,
		}, {
			Type: transaction.SpendOutputTypeMemoMessage,
			Data: []byte(message),
		}})
		if err != nil {
			r.Error(jerr.Get("error creating tx", err), http.StatusInternalServerError)
			return
		}

		fmt.Println(transaction.GetTxInfo(tx))
		node.BitcoinNode.Peer.QueueMessage(tx, nil)

		err = transaction.SaveTransaction(tx, nil)
		if err != nil {
			r.Error(jerr.Get("error saving transaction", err), http.StatusUnprocessableEntity)
			return
		}
	},
}