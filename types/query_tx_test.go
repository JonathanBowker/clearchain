package types

import (
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
	crypto "github.com/tendermint/go-crypto"
	tmsp "github.com/tendermint/tmsp/types"
)

func TestAccountQueryTx_String(t *testing.T) {
	type fields struct {
		Accounts  []string
		Address   []byte
		Signature crypto.Signature
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"stringRepr", fields{Accounts: []string{"account_1", "account_2"}, Address: []byte{byte(0x01)}}, "AccountQueryTx{01 [account_1 account_2]}"},
	}
	for _, tt := range tests {
		tx := AccountQueryTx{
			Accounts:  tt.fields.Accounts,
			Address:   tt.fields.Address,
			Signature: tt.fields.Signature,
		}
		if got := tx.String(); got != tt.want {
			t.Errorf("%q. AccountQueryTx.String() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAccountQueryTx_TxType(t *testing.T) {
	type fields struct {
		Accounts  []string
		Address   []byte
		Signature crypto.Signature
	}
	tests := []struct {
		name   string
		fields fields
		want   byte
	}{
		{"queryType", fields{}, TxTypeQueryAccount},
	}
	for _, tt := range tests {
		tx := AccountQueryTx{
			Accounts:  tt.fields.Accounts,
			Address:   tt.fields.Address,
			Signature: tt.fields.Signature,
		}
		if got := tx.TxType(); got != tt.want {
			t.Errorf("%q. AccountQueryTx.TxType() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAccountQueryTx_ValidateBasic(t *testing.T) {
	privKey := crypto.GenPrivKeyEd25519()
	pubKeyAddr := privKey.PubKey().Address()
	signature := privKey.Sign(pubKeyAddr)
	genUUID := func() string {
		return uuid.NewV4().String()
	}
	type fields struct {
		Accounts  []string
		Address   []byte
		Signature crypto.Signature
	}
	tests := []struct {
		name   string
		fields fields
		want   tmsp.Result
	}{
		{"invalidAddress", fields{[]string{}, []byte("addr"), nil}, tmsp.ErrBaseInvalidInput},
		{"invalidSignature", fields{[]string{}, pubKeyAddr, nil}, tmsp.ErrBaseInvalidSignature},
		{"emptyAccounts", fields{[]string{}, pubKeyAddr, signature}, tmsp.ErrBaseInvalidInput},
		{"invalidAccounts", fields{[]string{""}, pubKeyAddr, signature}, tmsp.ErrBaseInvalidInput},
		{"valid", fields{[]string{genUUID(), genUUID()}, pubKeyAddr, signature}, tmsp.OK},
	}
	for _, tt := range tests {
		tx := AccountQueryTx{
			Accounts:  tt.fields.Accounts,
			Address:   tt.fields.Address,
			Signature: tt.fields.Signature,
		}
		if got := tx.ValidateBasic(); got.Code != tt.want.Code {
			t.Errorf("%q. AccountQueryTx.ValidateBasic() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAccountQueryTx_SignBytes(t *testing.T) {
	chainID := "test_chain_id"
	privKey := crypto.GenPrivKeyEd25519()
	pubKeyAddr := privKey.PubKey().Address()
	accounts := []string{uuid.NewV4().String()}
	signBytes := func(accounts []string, addr []byte) []byte {
		return AccountQueryTx{Accounts: accounts, Address: addr}.SignBytes(chainID)
	}

	type fields struct {
		Accounts  []string
		Address   []byte
		Signature crypto.Signature
	}
	type args struct {
		chainID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{"signed", fields{accounts, pubKeyAddr, nil}, args{chainID}, signBytes(accounts, pubKeyAddr)},
	}
	for _, tt := range tests {
		tx := AccountQueryTx{
			Accounts:  tt.fields.Accounts,
			Address:   tt.fields.Address,
			Signature: tt.fields.Signature,
		}
		if got := tx.SignBytes(tt.args.chainID); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. AccountQueryTx.SignBytes() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAccountIndexQueryTx_ValidateBasic(t *testing.T) {
	privKey := crypto.GenPrivKeyEd25519()
	pubKeyAddr := privKey.PubKey().Address()
	signature := privKey.Sign(pubKeyAddr)
	type fields struct {
		Address   []byte
		Signature crypto.Signature
	}
	tests := []struct {
		name   string
		fields fields
		want   tmsp.Result
	}{
		{"invalidAddress", fields{[]byte("addr"), nil}, tmsp.ErrBaseInvalidInput},
		{"invalidSignature", fields{pubKeyAddr, nil}, tmsp.ErrBaseInvalidSignature},
		{"valid", fields{pubKeyAddr, signature}, tmsp.OK},
	}
	for _, tt := range tests {
		tx := AccountIndexQueryTx{
			Address:   tt.fields.Address,
			Signature: tt.fields.Signature,
		}
		if got := tx.ValidateBasic(); got.Code != tt.want.Code {
			t.Errorf("%q. AccountIndexQueryTx.ValidateBasic() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAccountIndexQueryTx_SignBytes(t *testing.T) {
	chainID := "test_chain_id"
	privKey := crypto.GenPrivKeyEd25519()
	pubKeyAddr := privKey.PubKey().Address()
	signBytes := func(addr []byte) []byte {
		return AccountIndexQueryTx{Address: addr}.SignBytes(chainID)
	}
	type fields struct {
		Address   []byte
		Signature crypto.Signature
	}
	type args struct {
		chainID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{"signed", fields{pubKeyAddr, nil}, args{chainID}, signBytes(pubKeyAddr)},
	}
	for _, tt := range tests {
		tx := AccountIndexQueryTx{
			Address:   tt.fields.Address,
			Signature: tt.fields.Signature,
		}
		if got := tx.SignBytes(tt.args.chainID); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. AccountIndexQueryTx.SignBytes() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
