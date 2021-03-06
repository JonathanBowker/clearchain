package types

// Perm is a synonym of uint64
type Perm uint64

// Each permission listed below reprent a respective transaction.
const (
	PermTransferTx = Perm(1 << iota)
	PermCreateAccountTx
	PermCreateLegalEntityTx
	PermCreateUserTx
	PermNone = Perm(0)
)

var permissionsMapByTxType = map[byte]Perm{
	TxTypeTransfer:          PermTransferTx,
	TxTypeCreateAccount:     PermCreateAccountTx,
	TxTypeCreateLegalEntity: PermCreateLegalEntityTx,
	TxTypeCreateUser:        PermCreateUserTx,
}

// NewPermByTxType creates a Perm object by ORing the Tx respective permissions.
func NewPermByTxType(bs ...byte) Perm {
	var p Perm
	for _, b := range bs {
		p = p.Add(permissionsMapByTxType[b])
	}
	return p
}

// Has returns (p & perms) != 0
func (p Perm) Has(perms Perm) bool {
	return (p & perms) != 0
}

// Add returns p | perms
func (p Perm) Add(perms Perm) Perm {
	return p | perms
}

// Clear returns p & (p ^ perms), in fact disabling p's bits given in perms.
func (p Perm) Clear(perms Perm) Perm {
	return p & ^perms
}
