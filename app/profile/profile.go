package profile

import (
	"bytes"
	"fmt"
	"git.jasonc.me/main/bitcoin/bitcoin/wallet"
	"git.jasonc.me/main/memo/app/db"
	"github.com/btcsuite/btcutil"
	"github.com/cpacia/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
)

type Profile struct {
	Name       string
	PkHash     []byte
	NameTx     []byte
	Self       bool
	SelfPkHash []byte
	Balance    int64
	BalanceBCH float64
	hasBalance bool
	Followers  []*Follower
	Following  []*Follower
}

func (p Profile) IsSelf() bool {
	return bytes.Equal(p.PkHash, p.SelfPkHash)
}

func (p Profile) CanFollow() bool {
	if p.IsSelf() {
		return false
	}
	for	_, follower := range p.Followers {
		if bytes.Equal(follower.PkHash, p.SelfPkHash) {
			return false
		}
	}
	return true
}

func (p Profile) CanUnFollow() bool {
	if p.IsSelf() {
		return false
	}
	for	_, follower := range p.Followers {
		if bytes.Equal(follower.PkHash, p.SelfPkHash) {
			return true
		}
	}
	return false
}

func (p Profile) HasBalance() bool {
	return p.hasBalance
}

func (p Profile) NameSet() bool {
	return len(p.NameTx) > 0
}

func (p Profile) GetNameTx() string {
	hash, err := chainhash.NewHash(p.NameTx)
	if err != nil {
		return ""
	}
	return hash.String()
}

func (p Profile) GetAddressString() string {
	addr, err := btcutil.NewAddressPubKeyHash(p.PkHash, &wallet.MainNetParamsOld)
	if err != nil {
		return ""
	}
	return addr.String()
}

func (p *Profile) SetBalances() error {
	transactions, err := db.GetTransactionsForPkHash(p.PkHash)
	if err != nil {
		return jerr.Get("error getting transactions for key", err)
	}
	var balance int64
	var balanceBCH float64
	for _, transaction := range transactions {
		for address, value := range transaction.GetValues() {
			if address == p.GetAddressString() {
				balance += value.GetValue()
				balanceBCH += value.GetValueBCH()
			}
		}
	}
	p.Balance = balance
	p.BalanceBCH = balanceBCH
	p.hasBalance = true
	return nil
}

func (p *Profile) SetFollowers() error {
	followers, err := GetFollowers(p.PkHash)
	if err != nil {
		return jerr.Get("error getting followers for hash", err)
	}
	p.Followers = followers
	return nil
}

func (p *Profile) SetFollowing() error {
	following, err := GetFollowing(p.PkHash)
	if err != nil {
		return jerr.Get("error getting following for hash", err)
	}
	p.Following = following
	return nil
}

func GetProfiles(selfPkHash []byte) ([]*Profile, error) {
	pkHashes, err := db.GetUniqueMemoAPkHashes()
	if err != nil {
		return nil, jerr.Get("error getting unique pk hashes", err)
	}
	var profiles []*Profile
	for _, pkHash := range pkHashes {
		profile, err := GetProfile(pkHash, selfPkHash)
		if err != nil {
			return nil, jerr.Get("error getting profile for hash", err)
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func GetProfileAndSetFollowers(pkHash []byte, selfPkHash []byte) (*Profile, error) {
	pf, err := GetProfile(pkHash, selfPkHash)
	if err != nil {
		return nil, jerr.Get("error getting profile for hash", err)
	}
	err = pf.SetFollowers()
	if err != nil {
		return nil, jerr.Get("error setting followers for profile", err)
	}
	return pf, nil
}

func GetProfile(pkHash []byte, selfPkHash []byte) (*Profile, error) {
	var name string
	var nameTx []byte
	memoSetName, err := db.GetNameForPkHash(pkHash)
	if err != nil && ! db.IsRecordNotFoundError(err) {
		return nil, jerr.Get("error getting MemoSetName for hash", err)
	}
	if memoSetName != nil {
		name = memoSetName.Name
		nameTx = memoSetName.TxHash
	}
	profile := &Profile{
		Name:       name,
		PkHash:     pkHash,
		NameTx:     nameTx,
		SelfPkHash: selfPkHash,
	}
	if profile.Name == "" {
		profile.Name = fmt.Sprintf("Profile %.6s", profile.GetAddressString())
	}
	return profile, nil
}

func GetProfileAndSetBalances(pkHash []byte, selfPkHash []byte) (*Profile, error) {
	pf, err := GetProfile(pkHash, selfPkHash)
	if err != nil {
		return nil, jerr.Get("error getting profile", err)
	}
	err = pf.SetBalances()
	if err != nil {
		return nil, jerr.Get("error setting balances", err)
	}
	return pf, nil
}
