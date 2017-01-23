package state

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/satori/go.uuid"
	abci "github.com/tendermint/abci/types"
	bctypes "github.com/tendermint/basecoin/types"
	bscoin "github.com/tendermint/basecoin/types"
	"github.com/tendermint/clearchain/testutil"
	"github.com/tendermint/clearchain/testutil/mocks/mock_account"
	"github.com/tendermint/clearchain/testutil/mocks/mock_user"
	"github.com/tendermint/clearchain/types"
	crypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-events"
)

func TestExecTx(t *testing.T) {
	// Set up fixtures
	chainID := "chain"
	s := NewState(bscoin.NewMemKVStore())
	s.chainID = chainID
	// Create users, legal entities and their respective accounts
	senderEntity := testutil.RandCH()
	randUsers := testutil.RandUsersWithLegalEntity(20, senderEntity, senderEntity.Permissions)
	senderUser := randUsers[0]
	recipientEntity := testutil.RandCustodian(senderUser.User.PubKey.Address())
	senderAccount := testutil.RandAccount(senderEntity)
	recipientAccount := testutil.RandAccount(recipientEntity)
	// Initialize the state
	s.SetLegalEntity(senderEntity.ID, senderEntity)
	s.SetLegalEntity(recipientEntity.ID, recipientEntity)
	s.SetAccount(senderAccount.ID, senderAccount)
	s.SetAccount(recipientAccount.ID, recipientAccount)
	for _, u := range randUsers {
		s.SetUser(u.User.PubKey.Address(), &u.User)
	}

	// Create a valid Tx without countersigners
	ccy := "USD"
	amount := int64(10000000)
	tx1 := func() types.TransferTx {
		tx := types.TransferTx{
			Sender: types.TxTransferSender{
				Address:   senderUser.User.PubKey.Address(),
				AccountID: senderAccount.ID,
				Amount:    amount,
				Currency:  ccy,
				Sequence:  1,
			},
			Recipient: types.TxTransferRecipient{
				AccountID: recipientAccount.ID,
			},
		}
		signBytes := tx.SignBytes(chainID)
		tx.Sender.Signature = senderUser.PrivKey.Sign(signBytes)
		return tx
	}()
	tx2 := func() types.TransferTx {
		// Create a valid Tx with countersigners
		counterSignersUsers := randUsers[2:]
		counterSigners := []types.TxTransferCounterSigner{}
		for _, u := range counterSignersUsers {
			cs := types.TxTransferCounterSigner{Address: u.User.PubKey.Address()}
			cs.SignTx(u.PrivKey, s.GetChainID())
			counterSigners = append(counterSigners, cs)
		}
		tx := types.TransferTx{
			Sender: types.TxTransferSender{
				Address:   senderUser.User.PubKey.Address(),
				AccountID: senderAccount.ID,
				Amount:    amount,
				Currency:  ccy,
				Sequence:  2,
			},
			CounterSigners: counterSigners,
			Recipient: types.TxTransferRecipient{
				AccountID: recipientAccount.ID,
			},
		}
		tx.SignTx(senderUser.PrivKey, s.GetChainID())

		return tx
	}()
	tx3 := func() types.CreateAccountTx {
		user := randUsers[0]
		tx := types.CreateAccountTx{
			Address:   user.User.PubKey.Address(),
			AccountID: uuid.NewV4().String(),
		}
		signBytes := tx.SignBytes(chainID)
		tx.Signature = user.Sign(signBytes)
		return tx
	}()
	tx4 := func() types.CreateLegalEntityTx {
		superEntity := testutil.RandCH()
		user := testutil.RandUsersWithLegalEntity(1, superEntity, superEntity.Permissions)[0]
		s.SetLegalEntity(superEntity.ID, superEntity)
		s.SetUser(user.User.PubKey.Address(), &user.User)
		tx := types.CreateLegalEntityTx{
			Address:  user.User.PubKey.Address(),
			EntityID: uuid.NewV4().String(),
			Type:     types.EntityTypeCustodianByte,
			Name:     "new Custodian",
			ParentID: uuid.NewV4().String(),
		}
		signBytes := tx.SignBytes(chainID)
		tx.Signature = user.Sign(signBytes)
		return tx
	}()
	tx5 := func() types.CreateLegalEntityTx {
		superEntity := testutil.RandCH()
		user := testutil.RandUsersWithLegalEntity(1, superEntity, superEntity.Permissions.Clear(types.PermCreateLegalEntityTx))[0]
		s.SetLegalEntity(superEntity.ID, superEntity)
		s.SetUser(user.User.PubKey.Address(), &user.User)
		tx := types.CreateLegalEntityTx{
			Address:  user.User.PubKey.Address(),
			EntityID: uuid.NewV4().String(),
			Type:     types.EntityTypeCustodianByte,
			Name:     "new Custodian",
			ParentID: uuid.NewV4().String(),
		}
		signBytes := tx.SignBytes(chainID)
		tx.Signature = user.Sign(signBytes)
		return tx
	}()
	tx6 := func() types.CreateUserTx {
		pubKey := crypto.GenPrivKeyEd25519().PubKey()
		entity := testutil.RandGCM([]byte{})
		user := testutil.RandUsersWithLegalEntity(1, entity, entity.Permissions)[0]
		s.SetLegalEntity(entity.ID, entity)
		s.SetUser(user.User.PubKey.Address(), &user.User)
		tx := types.CreateUserTx{
			Address:   user.User.PubKey.Address(),
			PubKey:    pubKey,
			CanCreate: true,
			Name:      "new user",
		}
		signBytes := tx.SignBytes(chainID)
		tx.Signature = user.Sign(signBytes)
		return tx
	}()
	tx7 := func() types.CreateUserTx {
		pubKey := crypto.GenPrivKeyEd25519().PubKey()
		entity := testutil.RandGCM([]byte{})
		user := testutil.RandUsersWithLegalEntity(1, entity, entity.Permissions.Clear(types.PermCreateUserTx))[0]
		s.SetLegalEntity(entity.ID, entity)
		s.SetUser(user.User.PubKey.Address(), &user.User)
		tx := types.CreateUserTx{
			Address:   user.User.PubKey.Address(),
			PubKey:    pubKey,
			CanCreate: true,
			Name:      "new user",
		}
		signBytes := tx.SignBytes(chainID)
		tx.Signature = user.Sign(signBytes)
		return tx
	}()
	tx8 := func() types.CreateUserTx {
		pubKey := crypto.GenPrivKeyEd25519().PubKey()
		entity := testutil.RandGCM([]byte{})
		user := testutil.RandUsersWithLegalEntity(1, entity, entity.Permissions)[0]
		s.SetLegalEntity(entity.ID, entity)
		s.SetUser(user.User.PubKey.Address(), &user.User)
		tx := types.CreateUserTx{
			Address:   user.User.PubKey.Address(),
			PubKey:    pubKey,
			CanCreate: false,
			Name:      "new user",
		}
		signBytes := tx.SignBytes(chainID)
		tx.Signature = user.Sign(signBytes)
		return tx
	}()
	type args struct {
		state     *State
		pgz       *bctypes.Plugins
		tx        types.Tx
		isCheckTx bool
		evc       events.Fireable
	}
	tests := []struct {
		name string
		args args
		want abci.Result
	}{
		{"checkTxTransferTxWithoutCounterSigners", args{s, nil, &tx1, true, nil}, abci.OK},
		{"appendTxTransferTxWithoutCounterSigners", args{s, nil, &tx1, false, nil}, abci.OK},
		{"checkTxTransferTxWithCounterSigners", args{s, nil, &tx2, true, nil}, abci.OK},
		{"appendTxTransferTxWithCounterSigners", args{s, nil, &tx2, false, nil}, abci.OK},
		{"checkTxCreateAccountTx", args{s, nil, &tx3, true, nil}, abci.OK},
		{"appendTxCreateAccountTx", args{s, nil, &tx3, false, nil}, abci.OK},
		{"checkTxCreateLegalEntityTx", args{s, nil, &tx4, true, nil}, abci.OK},
		{"appendTxCreateLegalEntityTx", args{s, nil, &tx4, false, nil}, abci.OK},
		{"checkTxCreateLegalEntityTxUnauthorized", args{s, nil, &tx5, true, nil}, abci.ErrUnauthorized},
		{"appendTxCreateLegalEntityTxUnauthorized", args{s, nil, &tx5, false, nil}, abci.ErrUnauthorized},
		{"checkTxCreateUserTx", args{s, nil, &tx6, true, nil}, abci.OK},
		{"appendTxCreateUserTx", args{s, nil, &tx6, false, nil}, abci.OK},
		{"checkTxCreateUserTxUnauthorized", args{s, nil, &tx7, true, nil}, abci.ErrUnauthorized},
		{"appendTxCreateUserTxUnauthorized", args{s, nil, &tx7, false, nil}, abci.ErrUnauthorized},
		{"checkTxCreateUserTx_CantCreate", args{s, nil, &tx8, true, nil}, abci.OK},
		{"appendTxCreateUserTx_CantCreate", args{s, nil, &tx8, false, nil}, abci.OK},
	}
	transferCnt := 0
	for _, tt := range tests {
		got := ExecTx(tt.args.state, tt.args.pgz, tt.args.tx, tt.args.isCheckTx, tt.args.evc)
		if got.Code != tt.want.Code {
			t.Errorf("%q. ExecTx() = %v, want %v", tt.name, got, tt.want)
		}
		switch tt.args.tx.(type) {
		case *types.TransferTx:
			if !tt.args.isCheckTx {
				senderAccount := s.GetAccount(senderAccount.ID)
				recipientAccount := s.GetAccount(recipientAccount.ID)
				senderWallet := senderAccount.GetWallet(ccy)
				recipientWallet := recipientAccount.GetWallet(ccy)
				transferCnt++

				balanceWanted := int64(transferCnt) * -amount
				if senderWallet.Balance != balanceWanted {
					t.Errorf("%q. senderWallet.Balance = %v, want %v", tt.name, senderWallet.Balance, balanceWanted)
				}
				if senderWallet.Sequence != transferCnt {
					t.Errorf("%q. senderWallet.Sequence = %v, want %v", tt.name, senderWallet.Sequence, transferCnt)
				}
				balanceWanted = int64(transferCnt) * amount
				if recipientWallet.Balance != balanceWanted {
					t.Errorf("%q. recipientWallet.Balance = %v, want %v", tt.name, recipientWallet.Balance, balanceWanted)
				}
				if recipientWallet.Sequence != transferCnt {
					t.Errorf("%q. recipientWallet.Sequence = %v, want %v", tt.name, recipientWallet.Sequence, transferCnt)
				}
			}
		case *types.CreateAccountTx:
			concreteTx := tt.args.tx.(*types.CreateAccountTx)
			if got.IsOK() && !tt.args.isCheckTx {
				if index := s.GetAccountIndex(); !index.Has(concreteTx.AccountID) {
					t.Errorf("%q. AccountIndex.Has(%s) = false, want true", tt.name, concreteTx.AccountID)
				}
				newAccount := s.GetAccount(concreteTx.AccountID)
				want := types.NewAccount(concreteTx.AccountID, randUsers[0].User.EntityID)
				if !newAccount.Equal(want) {
					t.Errorf("%q. created = %v, want %v", tt.name, newAccount, want)
				}
			}
			if got.IsOK() && tt.args.isCheckTx {
				if ret := s.GetAccount(concreteTx.AccountID); ret != nil {
					t.Errorf("%q. GetAccount(%q) = %v, want nil", tt.name, concreteTx.AccountID, ret)
				}
				if ret := s.GetAccountIndex(); ret != nil {
					t.Errorf("%q. GetAccountIndex() = %v, want nil", tt.name, ret)
				}
			}
		case *types.CreateLegalEntityTx:
			concreteTx := tt.args.tx.(*types.CreateLegalEntityTx)
			if got.IsOK() && !tt.args.isCheckTx {
				newEntity := s.GetLegalEntity(concreteTx.EntityID)
				want := types.NewLegalEntityByType(concreteTx.Type, concreteTx.EntityID, concreteTx.Name, concreteTx.Address, concreteTx.ParentID)
				if !newEntity.Equal(want) {
					t.Errorf("%q. created %v, want %v", tt.name, newEntity, want)
				}
			}
			if got.IsOK() && tt.args.isCheckTx {
				if ret := s.GetLegalEntity(concreteTx.EntityID); ret != nil {
					t.Errorf("%q. GetLegalEntity(%q) = %v, want nil", tt.name, concreteTx.EntityID, ret)
				}
			}
		case *types.CreateUserTx:
			concreteTx := tt.args.tx.(*types.CreateUserTx)
			newUserAddr := concreteTx.PubKey.Address()
			if got.IsOK() && !tt.args.isCheckTx {
				newUser := s.GetUser(newUserAddr)
				creator := s.GetUser(concreteTx.Address)
				perms := creator.Permissions
				if !concreteTx.CanCreate {
					perms = perms.Clear(types.PermCreateUserTx.Add(types.PermCreateLegalEntityTx))
				}
				want := types.NewUser(concreteTx.PubKey, concreteTx.Name, creator.EntityID, perms)
				if !newUser.Equal(want) {
					t.Errorf("%q. created %v, want %v", tt.name, newUser, want)
				}
			}
			if got.IsOK() && tt.args.isCheckTx {
				if ret := s.GetUser(newUserAddr); ret != nil {
					t.Errorf("%q. GetUser(%q) = %v, want nil", tt.name, newUserAddr, ret)
				}
			}
		}

	}
}

func TestExecQueryTx(t *testing.T) {
	// Set up fixtures
	chainID := "chain"
	s := NewState(bscoin.NewMemKVStore())
	s.chainID = chainID
	// Create users, legal entities and their respective accounts
	entity := testutil.RandCH()
	s.SetLegalEntity(entity.ID, entity)
	user := testutil.PrivUserWithLegalEntityFromSecret("", entity, entity.Permissions)
	// Initialize the state
	accountIndex := types.NewAccountIndex()
	s.SetUser(user.User.PubKey.Address(), &user.User)
	accounts := testutil.RandAccounts(10, entity)
	for _, account := range accounts {
		account.Wallets = []types.Wallet{
			testutil.RandWallet(types.Currencies["GBP"], 100000, 99999999),
			testutil.RandWallet(types.Currencies["EUR"], 100000, 99999999),
			testutil.RandWallet(types.Currencies["USD"], 100000, 99999999),
		}
		s.SetAccount(account.ID, account)
		accountIndex.Add(account.ID)
	}
	s.SetAccountIndex(accountIndex)

	accountIDs := make([]string, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.ID
	}
	// Create a valid QueryTx
	validAccountQueryTx := types.AccountQueryTx{
		Address:  user.User.PubKey.Address(),
		Accounts: accountIDs,
	}
	validAccountQueryTx.Signature = user.PrivKey.Sign(validAccountQueryTx.SignBytes(chainID))
	// Create a signed QueryTx with invalid accounts
	invalidAccountsQueryTx := types.AccountQueryTx{
		Address:  user.User.PubKey.Address(),
		Accounts: append(accountIDs, ""),
	}
	invalidAccountsQueryTx.Signature = user.PrivKey.Sign(invalidAccountsQueryTx.SignBytes(chainID))
	expectedJSON, _ := json.Marshal(struct {
		Account []*types.Account `json:"accounts"`
	}{Account: accounts})
	// Create a signed AccountIndexQueryTx
	validAccountIndexQueryTx := types.AccountIndexQueryTx{
		Address: user.User.PubKey.Address(),
	}
	validAccountIndexQueryTx.Signature = user.PrivKey.Sign(validAccountIndexQueryTx.SignBytes(chainID))
	validAccountIndexQueryTxExpectedJSON, _ := json.Marshal(accountIndex)

	type args struct {
		state *State
		tx    types.Tx
	}
	tests := []struct {
		name string
		args args
		want abci.Result
	}{
		{"queryAccount", args{s, &validAccountQueryTx}, abci.NewResultOK(expectedJSON, "")},
		{"invalidAccountID", args{s, &invalidAccountsQueryTx}, abci.ErrBaseInvalidInput},
		{"notExistingAccount", args{s, func(t types.AccountQueryTx) *types.AccountQueryTx {
			t.Accounts = append(t.Accounts, uuid.NewV4().String())
			return &t
		}(invalidAccountsQueryTx)}, abci.ErrBaseInvalidInput},
		{"queryAccountIndex", args{s, &validAccountIndexQueryTx}, abci.NewResultOK(validAccountIndexQueryTxExpectedJSON, "")},
	}
	for _, tt := range tests {
		got := ExecQueryTx(tt.args.state, tt.args.tx)
		if got.IsErr() && got.Code != tt.want.Code {
			t.Errorf("%q. ExecQueryTx() = %v, want %v", tt.name, got, tt.want)
		}
		if got.IsOK() && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. ExecQueryTx() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_validateSender(t *testing.T) {
	user := testutil.PrivUserFromSecret("")
	authorizedUser := &user.User
	authorizedUser.EntityID = uuid.NewV4().String()
	authorizedUser.Permissions = types.PermTransferTx
	authorizedLegalEntity := &types.LegalEntity{Permissions: types.PermTransferTx}
	validAccount := &types.Account{EntityID: authorizedUser.EntityID}
	validTx := &types.TransferTx{Sender: types.TxTransferSender{Address: authorizedUser.PubKey.Address()}}
	signBytes := validTx.SignBytes("chainID")
	validTx.Sender.Signature = user.PrivKey.Sign(signBytes)
	type args struct {
		acc       *types.Account
		entity    *types.LegalEntity
		u         *types.User
		signBytes []byte
		tx        *types.TransferTx
	}
	tests := []struct {
		name string
		args args
		want abci.Result
	}{
		{
			"unauthorizedUser",
			args{validAccount, authorizedLegalEntity, &types.User{}, signBytes, validTx},
			abci.ErrUnauthorized,
		},
		{
			"unauthorizedEntity",
			args{validAccount, &types.LegalEntity{}, authorizedUser, signBytes, validTx},
			abci.ErrUnauthorized,
		},
		{
			"invalidSignature",
			args{validAccount, authorizedLegalEntity, authorizedUser, []byte{}, validTx},
			abci.ErrBaseInvalidSignature,
		},
		{
			"invalidSignature",
			args{validAccount, authorizedLegalEntity, authorizedUser, signBytes, validTx},
			abci.OK,
		},
	}
	for _, tt := range tests {
		if got := validateSender(tt.args.acc, tt.args.entity, tt.args.u, tt.args.signBytes, tt.args.tx); got.Code != tt.want.Code {
			t.Errorf("%q. validateSender() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_validatePermissions(t *testing.T) {
	authorizedUser := &types.User{EntityID: uuid.NewV4().String(), Permissions: types.PermTransferTx}
	authorizedLegalEntity := &types.LegalEntity{Permissions: types.PermTransferTx}
	validAccount := &types.Account{EntityID: authorizedUser.EntityID}

	type args struct {
		u  *types.User
		e  *types.LegalEntity
		a  *types.Account
		tx *types.TransferTx
	}
	tests := []struct {
		name string
		args args
		want abci.Result
	}{
		{
			"unauthorizedUser",
			args{
				&types.User{}, &types.LegalEntity{}, validAccount, &types.TransferTx{},
			},
			abci.ErrUnauthorized,
		},
		{
			"legalEntityMismatch",
			args{
				authorizedUser, &types.LegalEntity{}, validAccount, &types.TransferTx{},
			},
			abci.ErrUnauthorized,
		},
		{
			"accountMismatch",
			args{
				authorizedUser, authorizedLegalEntity, &types.Account{}, &types.TransferTx{},
			},
			abci.ErrUnauthorized,
		},
		{
			"unauthorizedLegalEntity",
			args{
				authorizedUser, &types.LegalEntity{}, validAccount, &types.TransferTx{},
			},
			abci.ErrUnauthorized,
		},
		{
			"authorizedUser",
			args{
				authorizedUser, authorizedLegalEntity, validAccount, &types.TransferTx{},
			},
			abci.OK,
		},
	}
	for _, tt := range tests {
		if got := validatePermissions(tt.args.u, tt.args.e, tt.args.a, tt.args.tx); got.Code != tt.want.Code {
			t.Errorf("%q. validatePermissions() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_applyChangesToInput(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObj := mock_account.NewMockAccountSetter(mockCtrl)
	accountID := uuid.NewV4().String()
	genTxTransferSender := func() types.TxTransferSender {
		return types.TxTransferSender{AccountID: accountID, Amount: 15, Currency: "USD"}
	}

	type args struct {
		state     types.AccountSetter
		in        types.TxTransferSender
		acc       *types.Account
		isCheckTx bool
	}
	tests := []struct {
		name string
		args args
	}{
		{"appendTx", args{mockObj, genTxTransferSender(), &types.Account{ID: accountID}, false}},
		{"checkTx", args{mockObj, genTxTransferSender(), &types.Account{ID: accountID}, true}},
	}
	for _, tt := range tests {
		ntimes := 0
		if !tt.args.isCheckTx {
			ntimes = 1
		}
		mockObj.EXPECT().SetAccount(tt.args.in.AccountID, tt.args.acc).Times(ntimes)
		applyChangesToInput(tt.args.state, tt.args.in, tt.args.acc, tt.args.isCheckTx)
	}
}

func Test_applyChangesToOutput(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObj := mock_account.NewMockAccountSetter(mockCtrl)
	genTxTransferSender := func() types.TxTransferSender {
		return types.TxTransferSender{Amount: 100, Currency: "USD"}
	}
	accountID := uuid.NewV4().String()
	genTxTransferRecipient := func() types.TxTransferRecipient {
		return types.TxTransferRecipient{AccountID: accountID}
	}
	type args struct {
		state     types.AccountSetter
		in        types.TxTransferSender
		out       types.TxTransferRecipient
		acc       *types.Account
		isCheckTx bool
	}
	tests := []struct {
		name string
		args args
	}{
		{"appendTx", args{mockObj, genTxTransferSender(), genTxTransferRecipient(), &types.Account{ID: accountID}, false}},
		{"checkTx", args{mockObj, genTxTransferSender(), genTxTransferRecipient(), &types.Account{ID: accountID}, true}},
	}
	for _, tt := range tests {
		ntimes := 0
		if !tt.args.isCheckTx {
			ntimes = 1
		}
		mockObj.EXPECT().SetAccount(tt.args.out.AccountID, tt.args.acc).Times(ntimes)
		applyChangesToOutput(tt.args.state, tt.args.in, tt.args.out, tt.args.acc, tt.args.isCheckTx)
	}
}

func Test_validateCounterSignersAdvanced(t *testing.T) {
	// Set up fixtures
	s := NewState(bscoin.NewMemKVStore())
	s.SetChainID("chainID")
	ent := &types.LegalEntity{
		ID:          uuid.NewV4().String(),
		Permissions: types.PermTransferTx}
	s.SetLegalEntity(ent.ID, ent)
	acc := &types.Account{ID: uuid.NewV4().String(), EntityID: ent.ID}
	s.SetAccount(acc.ID, acc)
	users := func(entity *types.LegalEntity) []*types.PrivUser {
		privUsers := testutil.RandUsers(10)
		usersSlice := make([]*types.PrivUser, 10)
		for i, u := range privUsers {
			u.User.EntityID = entity.ID
			u.User.Permissions = entity.Permissions
			usersSlice[i] = u
		}
		return usersSlice
	}(ent)
	senderprivateKey:=crypto.GenPrivKeyEd25519()
	transferTx := types.TransferTx{
		Sender: types.TxTransferSender{
			Address: senderprivateKey.PubKey().Address()},
		CounterSigners: make([]types.TxTransferCounterSigner, 10)}
	for i, u := range users {
		s.SetUser(u.User.PubKey.Address(), &u.User)
		transferTx.CounterSigners[i] = types.TxTransferCounterSigner{
			Address: u.User.PubKey.Address()}
		transferTx.CounterSigners[i].SignTx(u.PrivKey, s.GetChainID())
	}
	transferTx.SignTx(senderprivateKey, s.GetChainID())

	wrongSignatureTransferTx := types.TransferTx{
		Sender: types.TxTransferSender{
			Address: senderprivateKey.PubKey().Address()},
		CounterSigners: make([]types.TxTransferCounterSigner, 10)}
	for i, u := range users {
		s.SetUser(u.User.PubKey.Address(), &u.User)
		wrongSignatureTransferTx.CounterSigners[i] = types.TxTransferCounterSigner{
			Address: u.User.PubKey.Address()}
		wrongSignatureTransferTx.CounterSigners[i].SignTx(u.PrivKey, s.GetChainID())
	}
	wrongSignatureTransferTx.SignTx(senderprivateKey, s.GetChainID())
	wrongSignatureTransferTx.CounterSigners[0].Signature = senderprivateKey.Sign([]byte("wrong_bytes"))

	// Make sender a duplicate of a countersigner
	dupSendertransferTx := types.TransferTx{
		Sender: types.TxTransferSender{
			Address: transferTx.CounterSigners[0].Address},
		CounterSigners: transferTx.CounterSigners}
	// Non-existing user
	nonExistUserSendertransferTx := types.TransferTx{
		Sender: types.TxTransferSender{Address: crypto.CRandBytes(20)},
		CounterSigners: []types.TxTransferCounterSigner{
			types.TxTransferCounterSigner{Address: []byte("non-existing")}}}

	type args struct {
		state     *State
		acc       *types.Account
		entity    *types.LegalEntity
		tx        *types.TransferTx
	}
	tests := []struct {
		name string
		args args
		want abci.Result
	}{
		{"invalidUser",
			args{s, &types.Account{}, ent, &nonExistUserSendertransferTx},
			tmsp.ErrBaseUnknownAddress,
		},
		{"duplicateAddress",
			args{s, &types.Account{}, ent, &dupSendertransferTx},
			tmsp.ErrBaseDuplicateAddress,
		},
		{"invalidAccount",
			args{s, &types.Account{}, ent, &transferTx},
			tmsp.ErrUnauthorized,
		},
		{"invalidSignatures",
			args{s, acc, ent, &wrongSignatureTransferTx},
			tmsp.ErrBaseInvalidSignature,
		},
		{"validCounterSigners",
			args{s, acc, ent, &transferTx},
			tmsp.OK,
		},
	}
	for _, tt := range tests {
		if got := validateCounterSigners(tt.args.state, tt.args.acc, tt.args.entity, tt.args.tx); got.Code != tt.want.Code {
			t.Errorf("%q. validateCounterSigners() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_validateWalletSequence(t *testing.T) {
	type args struct {
		acc *types.Account
		in  types.TxTransferSender
	}
	tests := []struct {
		name string
		args args
		want abci.Result
	}{
		{"invalidInitialSequence", args{&types.Account{Wallets: []types.Wallet{}}, types.TxTransferSender{Currency: "USD", Sequence: 10}}, abci.ErrBaseInvalidSequence},
		{"invalidSequence", args{&types.Account{Wallets: []types.Wallet{types.Wallet{Currency: "USD", Sequence: 10}}}, types.TxTransferSender{Currency: "USD", Sequence: 10}}, abci.ErrBaseInvalidSequence},
		{"validInitialSequence", args{&types.Account{}, types.TxTransferSender{Currency: "USD", Sequence: 1}}, abci.OK},
		{"validSequence", args{&types.Account{Wallets: []types.Wallet{types.Wallet{Currency: "USD", Sequence: 10}}}, types.TxTransferSender{Currency: "USD", Sequence: 11}}, abci.OK},
	}
	for _, tt := range tests {
		if got := validateWalletSequence(tt.args.acc, tt.args.in); got.Code != tt.want.Code {
			t.Errorf("%q. validateWalletSequence() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_makeNewUser(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObj := mock_user.NewMockUserSetter(mockCtrl)
	user := testutil.RandUsers(1)[0]
	pubKey := crypto.GenPrivKeyEd25519().PubKey()
	genCreateUserTx := func() *types.CreateUserTx { return &types.CreateUserTx{Name: "Name", PubKey: pubKey, CanCreate: false} }
	type args struct {
		state     types.UserSetter
		creator   *types.User
		tx        *types.CreateUserTx
		isCheckTx bool
	}
	tests := []struct {
		name string
		args args
	}{
		{"appendTx", args{mockObj, &user.User, genCreateUserTx(), false}},
		{"checkTx", args{mockObj, &user.User, genCreateUserTx(), true}},
	}
	for _, tt := range tests {
		u := types.NewUser(tt.args.tx.PubKey, tt.args.tx.Name, tt.args.creator.EntityID, tt.args.creator.Permissions)
		if !tt.args.tx.CanCreate {
			u.Permissions = u.Permissions.Clear(types.PermCreateAccountTx.Add(types.PermCreateLegalEntityTx))
		}
		ntimes := 0
		if !tt.args.isCheckTx {
			ntimes = 1
		}
		mockObj.EXPECT().SetUser(tt.args.tx.PubKey.Address(), u).Times(ntimes)
		makeNewUser(tt.args.state, tt.args.creator, tt.args.tx, tt.args.isCheckTx)
	}
}
