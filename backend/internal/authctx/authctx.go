// Package authctx transporte l'ID de l'utilisateur authentifié à travers le
// contexte de la requête, depuis le middleware requireAuth (qui valide déjà
// la session) jusqu'aux handlers qui ont besoin de savoir qui appelle.
package authctx

import "context"

type contextKey int

const userIDKey contextKey = iota

func WithUserID(ctx context.Context, userID uint64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserID(ctx context.Context) (uint64, bool) {
	userID, ok := ctx.Value(userIDKey).(uint64)
	return userID, ok
}
