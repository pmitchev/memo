package key

import (
	"git.jasonc.me/main/memo/app/auth"
	"git.jasonc.me/main/memo/app/db"
	"git.jasonc.me/main/memo/app/res"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/web"
	"net/http"
	"strings"
)

var deleteAccountRoute = web.Route{
	Pattern:    res.UrlKeyDeleteAccount,
	NeedsLogin: true,
	Handler: func(r *web.Response) {
		r.Render()
	},
}

var deleteAccountSubmitRoute = web.Route{
	Pattern:     res.UrlKeyDeleteAccountSubmit,
	CsrfProtect: true,
	NeedsLogin:  true,
	Handler: func(r *web.Response) {
		user, err := auth.GetSessionUser(r.Session.CookieId)
		if err != nil {
			r.Error(jerr.Get("error getting session user", err), http.StatusInternalServerError)
			return
		}

		password := r.Request.GetFormValue("password")
		confirm := r.Request.GetFormValue("confirm")

		if strings.ToLower(confirm) != "delete account" {
			r.Error(jerr.New("delete account confirmation did not match"), http.StatusUnprocessableEntity)
		}

		key, err := db.GetKeyForUser(user.Id)
		if err != nil {
			r.Error(jerr.Get("error getting key", err), http.StatusInternalServerError)
			return
		}
		_, err = key.GetPrivateKey(password)
		if err != nil {
			r.Error(jerr.Get("error unlocking key, password doesn't match", err), http.StatusUnprocessableEntity)
			return
		}
		err = key.Delete()
		if err != nil {
			r.Error(jerr.Get("error deleting key", err), http.StatusUnauthorized)
			return
		}
		err = auth.DeleteAccount(user.Id, password)
		if err != nil {
			r.Error(jerr.Get("error deleting account", err), http.StatusUnauthorized)
			return
		}
		r.ResetOrCreateSession()
		r.Render()
	},
}
