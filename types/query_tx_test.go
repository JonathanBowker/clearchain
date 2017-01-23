package types

import (
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
	abci "github.com/tendermint/abci/types"
	crypto "github.com/tendermint/go-crypto"
)

func TestAccountQueryTx_String(t *testing.T) {
	signature, _ := crypto.SignatureFromBytes([]byte{1, 100, 140, 5, 246, 69, 107, 210, 41, 250, 189, 162, 44, 49, 6, 222, 185, 227, 247, 12, 213, 215, 246, 182, 66, 0, 233, 54, 215, 124, 175, 172, 235, 72, 151, 154, 26, 65, 145, 127, 121, 223, 4, 233, 210, 18, 188, 144, 72, 18, 63, 80, 158, 68, 221, 110, 82, 249, 26, 46, 202, 154, 43, 1, 13})

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
		{"stringRepr", fields{Accounts: []string{"account_1", "account_2"}, Address: []byte{byte(0x01)}, Signature: signature}, "AccountQueryTx{01 [account_1 account_2] /648C05F6456B.../}"},
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
		want   abci.Result
	}{
		{"invalidAddress", fields{[]string{}, []byte("addr"), nil}, abci.ErrBaseInvalidInput},
		{"invalidSignature", fields{[]string{}, pubKeyAddr, nil}, abci.ErrBaseInvalidSignature},
		{"emptyAccounts", fields{[]string{}, pubKeyAddr, signature}, abci.ErrBaseInvalidInput},
		{"invalidAccounts", fields{[]string{""}, pubKeyAddr, signature}, abci.ErrBaseInvalidInput},
		{"valid", fields{[]string{genUUID(), genUUID()}, pubKeyAddr, signature}, abci.OK},
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
		want   abci.Result
	}{
		{"invalidAddress", fields{[]byte("addr"), nil}, abci.ErrBaseInvalidInput},
		{"invalidSignature", fields{pubKeyAddr, nil}, abci.ErrBaseInvalidSignature},
		{"valid", fields{pubKeyAddr, signature}, abci.OK},
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

func TestAccountQueryTx_SignTx(t *testing.T) {
	privKey := crypto.GenPrivKeyEd25519()
	addr := privKey.PubKey().Address()
	type fields struct {
		Accounts  []string
		Address   []byte
		Signature crypto.Signature
	}
	type args struct {
		privateKey crypto.PrivKey
		chainID    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"validSignature", fields{[]string{"account_1"}, addr, nil}, args{privKey, "chainID"}, false},
		{"invalidSignature", fields{[]string{"account_1"}, addr, nil}, args{crypto.GenPrivKeyEd25519(), "chainID"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := AccountQueryTx{
				Accounts:  tt.fields.Accounts,
				Address:   tt.fields.Address,
				Signature: tt.fields.Signature,
			}
			if err := tx.SignTx(tt.args.privateKey, tt.args.chainID); (err != nil) != tt.wantErr {
				t.Errorf("AccountQueryTx.SignTx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccountIndexQueryTx_SignTx(t *testing.T) {
	privKey := crypto.GenPrivKeyEd25519()
	addr := privKey.PubKey().Address()

	type fields struct {
		Address   []byte
		Signature crypto.Signature
	}
	type args struct {
		privateKey crypto.PrivKey
		chainID    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"validSignature", fields{addr, nil}, args{privKey, "chainID"}, false},
		{"invalidSignature", fields{addr, nil}, args{crypto.GenPrivKeyEd25519(), "chainID"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := AccountIndexQueryTx{
				Address:   tt.fields.Address,
				Signature: tt.fields.Signature,
			}
			if err := tx.SignTx(tt.args.privateKey, tt.args.chainID); (err != nil) != tt.wantErr {
				t.Errorf("AccountIndexQueryTx.SignTx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
