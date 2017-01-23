package state

import (
	"encoding/json"
	"fmt"

	abci "github.com/tendermint/abci/types"
	bctypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/clearchain/types"
	"github.com/tendermint/go-common"
	"github.com/tendermint/go-events"
)

func transfer(state *State, tx *types.TransferTx, isCheckTx bool) abci.Result {
	// // Validate basic structure
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	// Retrieve Sender's data
	user := state.GetUser(tx.Sender.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog("Sender's user is unknown")
	}
	entity := state.GetLegalEntity(user.EntityID)
	if entity == nil {
		return abci.ErrUnauthorized.AppendLog("User's does not belong to any LegalEntity")
	}

	// Get the accounts
	senderAccount := state.GetAccount(tx.Sender.AccountID)
	if senderAccount == nil {
		return abci.ErrBaseUnknownAddress.AppendLog("Sender's account is unknown")
	}
	recipientAccount := state.GetAccount(tx.Recipient.AccountID)
	if recipientAccount == nil {
		return abci.ErrBaseUnknownAddress.AppendLog("Unknown recipient address")
	}

	// Validate sender's Account
	if res := validateWalletSequence(senderAccount, tx.Sender); res.IsErr() {
		return res.PrependLog("in validateWalletSequence()")
	}

	// Generate byte-to-byte signature
	signBytes := tx.SignBytes(state.GetChainID())

	// Validate sender's permissions and signature
	if res := validateSender(senderAccount, entity, user, signBytes, tx); res.IsErr() {
		return res.PrependLog("in validateSender()")
	}
	// Validate counter signers
	if res := validateCounterSigners(state, senderAccount, entity, tx); res.IsErr() {
		return res.PrependLog("in validateCounterSigners()")
	}

	// Apply changes
	applyChangesToInput(state, tx.Sender, senderAccount, isCheckTx)
	applyChangesToOutput(state, tx.Sender, tx.Recipient, recipientAccount, isCheckTx)

	return abci.OK

}

func createAccount(state *State, tx *types.CreateAccountTx, isCheckTx bool) abci.Result {
	// // Validate basic structure
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	// Retrieve user data
	user := state.GetUser(tx.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog("User is unknown")
	}
	entity := state.GetLegalEntity(user.EntityID)
	if entity == nil {
		return abci.ErrUnauthorized.AppendLog("User's does not belong to any LegalEntity")
	}
	// Validate permissions
	if !types.CanExecTx(user, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"User is not authorized to execute the Tx: %s", user.String()))
	}
	if !types.CanExecTx(entity, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"LegalEntity is not authorized to execute the Tx: %s", entity.String()))
	}
	// Generate byte-to-byte signature and validate the signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !user.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrBaseInvalidSignature.AppendLog("user's signature doesn't match")
	}

	// Create the new account
	if acc := state.GetAccount(tx.AccountID); acc != nil {
		return abci.ErrBaseInvalidInput.AppendLog(common.Fmt("Account already exists: %q", tx.AccountID))
	}
	// Get or create the accounts index
	if !isCheckTx {
		acc := types.NewAccount(tx.AccountID, entity.ID)
		state.SetAccount(acc.ID, acc)
		return SetAccountInIndex(state, *acc)
	}

	return abci.OK
}

func createLegalEntity(state *State, tx *types.CreateLegalEntityTx, isCheckTx bool) abci.Result {
	// // Validate basic structure
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	// Retrieve user data
	user := state.GetUser(tx.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog("User is unknown")
	}
	entity := state.GetLegalEntity(user.EntityID)
	if entity == nil {
		return abci.ErrUnauthorized.AppendLog("User's does not belong to any LegalEntity")
	}
	// Validate permissions
	if !types.CanExecTx(user, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"User is not authorized to execute the Tx: %s", user.String()))
	}
	if !types.CanExecTx(entity, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"LegalEntity is not authorized to execute the Tx: %s", entity.String()))
	}
	// Generate byte-to-byte signature and validate the signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !user.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrBaseInvalidSignature.AppendLog("user's signature doesn't match")
	}

	// Create new legal entity
	if ent := state.GetLegalEntity(tx.EntityID); ent != nil {
		return abci.ErrBaseInvalidInput.AppendLog(common.Fmt("LegalEntity already exists: %q", tx.EntityID))
	}
	if !isCheckTx {
		legalEntity := types.NewLegalEntityByType(tx.Type, tx.EntityID, tx.Name, user.PubKey.Address(), tx.ParentID)
		state.SetLegalEntity(legalEntity.ID, legalEntity)
		return SetLegalEntityInIndex(state, legalEntity)
	}

	return abci.OK
}

func createUser(state *State, tx *types.CreateUserTx, isCheckTx bool) abci.Result {
	// // Validate basic structure
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	// Retrieve user data
	creator := state.GetUser(tx.Address)
	if creator == nil {
		return abci.ErrBaseUnknownAddress.AppendLog("User is unknown")
	}
	entity := state.GetLegalEntity(creator.EntityID)
	if entity == nil {
		return abci.ErrUnauthorized.AppendLog("User's does not belong to any LegalEntity")
	}

	// Validate permissions
	if !types.CanExecTx(creator, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"User is not authorized to execute the Tx: %s", creator.String()))
	}
	if !types.CanExecTx(entity, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"LegalEntity is not authorized to execute the Tx: %s", entity.String()))
	}
	// Generate byte-to-byte signature and validate the signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !creator.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrBaseInvalidSignature.AppendLog("user's signature doesn't match")
	}
	// Create new user
	if usr := state.GetUser(tx.PubKey.Address()); usr != nil {
		return abci.ErrBaseDuplicateAddress.AppendLog(common.Fmt("User already exists: %q", tx.PubKey.Address()))
	}
	makeNewUser(state, creator, tx, isCheckTx)

	return abci.OK
}

// ExecTx actually executes a Tx
func ExecTx(state *State, pgz *bctypes.Plugins, tx types.Tx,
	isCheckTx bool, evc events.Fireable) abci.Result {

	// Execute transaction
	switch tx := tx.(type) {
	case *types.TransferTx:
		return transfer(state, tx, isCheckTx)

	case *types.CreateAccountTx:
		return createAccount(state, tx, isCheckTx)

	case *types.CreateLegalEntityTx:
		return createLegalEntity(state, tx, isCheckTx)

	case *types.CreateUserTx:
		return createUser(state, tx, isCheckTx)

	default:
		return abci.ErrBaseEncodingError.SetLog("Unknown tx type")
	}
}

func accountQuery(state *State, tx *types.AccountQueryTx) abci.Result {
	// Validate basic
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	user := state.GetUser(tx.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog(common.Fmt("Address is unknown: %v", tx.Address))
	}
	accounts := make([]*types.Account, len(tx.Accounts))
	for i, accountID := range tx.Accounts {
		account := state.GetAccount(accountID)
		if account == nil {
			return abci.ErrBaseInvalidInput.AppendLog(common.Fmt("Invalid account_id: %q", accountID))
		}
		accounts[i] = account
	}

	// Generate byte-to-byte signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !user.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrUnauthorized.AppendLog("signature doesn't match")
	}
	data, err := json.Marshal(types.AccountsReturned{Account: accounts})
	if err != nil {
		return abci.ErrInternalError.AppendLog(common.Fmt("Couldn't make the response: %v", err))
	}
	return abci.OK.SetData(data)
}

func accountIndexQuery(state *State, tx *types.AccountIndexQueryTx) abci.Result {
	// Validate basic
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	user := state.GetUser(tx.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog(common.Fmt("Address is unknown: %v", tx.Address))
	}

	// Check that the account index exists
	accountIndex := state.GetAccountIndex()
	if accountIndex == nil {
		return abci.ErrInternalError.AppendLog("AccountIndex has not yet been initialized")
	}

	// Generate byte-to-byte signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !user.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrUnauthorized.AppendLog("signature doesn't match")
	}
	data, err := json.Marshal(accountIndex)
	if err != nil {
		return abci.ErrInternalError.AppendLog(common.Fmt("Couldn't make the response: %v", err))
	}
	return abci.OK.SetData(data)
}

// ExecQueryTx handles queries.
func ExecQueryTx(state *State, tx types.Tx) abci.Result {

	// Execute transaction
	switch tx := tx.(type) {
	case *types.AccountQueryTx:
		return accountQuery(state, tx)

	case *types.AccountIndexQueryTx:
		return accountIndexQuery(state, tx)

	case *types.LegalEntityQueryTx:
		return legalEntityQuery(state, tx)

	case *types.LegalEntityIndexQueryTx:
		return legalEntityIndexQueryTx(state, tx)

	default:
		return abci.ErrBaseEncodingError.SetLog("Unknown tx type")
	}
}

//--------------------------------------------------------------------------------

func validateWalletSequence(acc *types.Account, in types.TxTransferSender) abci.Result {
	wal := acc.GetWallet(in.Currency)
	// Wallet does not exist, Sequence must be 1
	if wal == nil {
		if in.Sequence != 1 {
			return abci.ErrBaseInvalidSequence.AppendLog(common.Fmt("Invalid sequence: got: %v, want: 1", in.Sequence))
		}
		return abci.OK
	}
	if in.Sequence != wal.Sequence+1 {
		return abci.ErrBaseInvalidSequence.AppendLog(common.Fmt("Invalid sequence: got: %v, want: %v", in.Sequence, wal.Sequence+1))
	}
	return abci.OK
}

func validateSender(acc *types.Account, entity *types.LegalEntity, u *types.User, signBytes []byte, tx *types.TransferTx) abci.Result {
	if res := validatePermissions(u, entity, acc, tx); res.IsErr() {
		return res
	}
	if !u.VerifySignature(signBytes, tx.Sender.Signature) {
		return abci.ErrBaseInvalidSignature.AppendLog("sender's signature doesn't match")
	}
	return abci.OK
}

// Validate countersignatures
func validateCounterSigners(state *State, acc *types.Account, entity *types.LegalEntity, tx *types.TransferTx) abci.Result {
	var users = make(map[string]bool)

	// Make sure users are not duplicated
	users[string(tx.Sender.Address)] = true

	for _, in := range tx.CounterSigners {
		// Users must not be duplicated either
		if _, ok := users[string(in.Address)]; ok {
			return abci.ErrBaseDuplicateAddress
		}
		users[string(in.Address)] = true

		// User must exist
		user := state.GetUser(in.Address)
		if user == nil {
			return abci.ErrBaseUnknownAddress
		}

		// Validate the permissions
		if res := validatePermissions(user, entity, acc, tx); res.IsErr() {
			return res
		}
		// Verify the signature
		if !user.VerifySignature(in.SignBytes(state.GetChainID()), in.Signature) {
			return abci.ErrBaseInvalidSignature.AppendLog(common.Fmt("countersigner's signature doesn't match, user: %s", user))
		}
	}

	return abci.OK
}

func validatePermissions(u *types.User, e *types.LegalEntity, a *types.Account, tx types.Tx) abci.Result {
	// Verify user belongs to the legal entity
	if !a.BelongsTo(u.EntityID) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"Access forbidden for user %s to account %s", u.Name, a.String()))
	}
	// Valdate permissions
	if !types.CanExecTx(u, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"User is not authorized to execute the Tx: %s", u.String()))
	}
	if !types.CanExecTx(e, tx) {
		return abci.ErrUnauthorized.AppendLog(common.Fmt(
			"LegalEntity is not authorized to execute the Tx: %s", e.String()))
	}
	return abci.OK
}

// Apply changes to inputs
func applyChangesToInput(state types.AccountSetter, in types.TxTransferSender, account *types.Account, isCheckTx bool) {
	applyChanges(account, in.Currency, in.Amount, false)

	if !isCheckTx {
		state.SetAccount(account.ID, account)
	}
}

// Apply changes to outputs
func applyChangesToOutput(state types.AccountSetter, in types.TxTransferSender, out types.TxTransferRecipient, account *types.Account, isCheckTx bool) {
	applyChanges(account, in.Currency, in.Amount, true)

	if !isCheckTx {
		state.SetAccount(account.ID, account)
	}
}

func applyChanges(account *types.Account, currency string, amount int64, isBuy bool) {

	wal := account.GetWallet(currency)

	if wal == nil {
		wal = &types.Wallet{Currency: currency}
	}

	if isBuy {
		wal.Balance += amount
	} else {
		wal.Balance += -amount
	}

	wal.Sequence++

	account.SetWallet(*wal)
}

func makeNewUser(state types.UserSetter, creator *types.User, tx *types.CreateUserTx, isCheckTx bool) {
	perms := creator.Permissions
	if !tx.CanCreate {
		perms = perms.Clear(types.PermCreateUserTx.Add(types.PermCreateLegalEntityTx))
	}
	user := types.NewUser(tx.PubKey, tx.Name, creator.EntityID, perms)
	if user == nil {
		common.PanicSanity(common.Fmt("Unexpected nil User"))
	}
	if !isCheckTx {
		state.SetUser(tx.PubKey.Address(), user)
	}
}

//Returns existing AccountIndex from store or creates new empty one
func GetOrMakeAccountIndex(state types.AccountIndexGetter) *types.AccountIndex {
	if index := state.GetAccountIndex(); index != nil {
		return index
	}
	return types.NewAccountIndex()
}

//Sets Account in AccountIndex in store
func SetAccountInIndex(state *State, account types.Account) abci.Result {
	accountIndex := GetOrMakeAccountIndex(state)
	if accountIndex.Has(account.ID) {
		return abci.ErrBaseInvalidInput.AppendLog(common.Fmt("Account already exists in the account index: %q", account.ID))
	}
	accountIndex.Add(account.ID)
	state.SetAccountIndex(accountIndex)
	return abci.OK
}

//Sets LegalEntity in LegalEntityIndex in store
func SetLegalEntityInIndex(state *State, legalEntity *types.LegalEntity) abci.Result {
	legalEntities := state.GetLegalEntityIndex()

	if legalEntities == nil {
		legalEntities = &types.LegalEntityIndex{Ids: []string{}}
	}

	if legalEntities.Has(legalEntity.ID) {
		return abci.ErrBaseInvalidInput.AppendLog(common.Fmt("LegalEntity already exists in the LegalEntity index: %q", legalEntity.ID))
	}
	legalEntities.Add(legalEntity.ID)

	state.SetLegalEntityIndex(legalEntities)

	return abci.OK
}

func legalEntityQuery(state *State, tx *types.LegalEntityQueryTx) abci.Result {
	// Validate basic
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	user := state.GetUser(tx.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog(common.Fmt("Address is unknown: %v", tx.Address))
	}
	legalEntities := make([]*types.LegalEntity, len(tx.Ids))
	for i, id := range tx.Ids {
		legalEntity := state.GetLegalEntity(id)
		if legalEntity == nil {
			return abci.ErrBaseInvalidInput.AppendLog(common.Fmt("Invalid legalEntity id: %q", id))
		}
		legalEntities[i] = legalEntity
	}

	// Generate byte-to-byte signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !user.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrUnauthorized.AppendLog("signature doesn't match")
	}
	data, err := json.Marshal(types.LegalEntitiesReturned{LegalEntities: legalEntities})
	if err != nil {
		return abci.ErrInternalError.AppendLog(common.Fmt("Couldn't make the response: %v", err))
	}
	return abci.OK.SetData(data)
}

func legalEntityIndexQueryTx(state *State, tx *types.LegalEntityIndexQueryTx) abci.Result {
	// Validate basic
	if res := tx.ValidateBasic(); res.IsErr() {
		return res.PrependLog("in ValidateBasic()")
	}

	user := state.GetUser(tx.Address)
	if user == nil {
		return abci.ErrBaseUnknownAddress.AppendLog(common.Fmt("Address is unknown: %v", tx.Address))
	}

	// Check that the account index exists
	legalEntities := state.GetLegalEntityIndex()
	if legalEntities == nil {
		return abci.ErrInternalError.AppendLog("LegalEntities has not yet been initialized")
	}

	// Generate byte-to-byte signature
	signBytes := tx.SignBytes(state.GetChainID())
	if !user.VerifySignature(signBytes, tx.Signature) {
		return abci.ErrUnauthorized.AppendLog("signature doesn't match")
	}
	data, err := json.Marshal(legalEntities)
	if err != nil {
		return abci.ErrInternalError.AppendLog(common.Fmt("Couldn't make the response: %v", err))
	}
	return abci.OK.SetData(data)
}
