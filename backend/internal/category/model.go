package category

type Type string

const (
	TypeAchat  Type = "achat"
	TypeRevenu Type = "revenu"
)

type Category struct {
	ID        uint64 `json:"id"`
	AccountID uint64 `json:"account_id"`
	Name      string `json:"name"`
	Type      Type   `json:"type"`
}
