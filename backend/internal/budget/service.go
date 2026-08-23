package budget

import (
	"context"

	"finance/backend/internal/category"
	"finance/backend/internal/incomes"
	"finance/backend/internal/purchase"
)

type Service struct {
	repo         *Repository
	categoryRepo *category.Repository
	purchaseRepo *purchase.Repository
	incomeRepo   *incomes.Repository
}

func NewService(repo *Repository, categoryRepo *category.Repository, purchaseRepo *purchase.Repository, incomeRepo *incomes.Repository) *Service {
	return &Service{repo: repo, categoryRepo: categoryRepo, purchaseRepo: purchaseRepo, incomeRepo: incomeRepo}
}

// EffectiveBudgets renvoie, pour chaque catégorie du compte, son budget
// effectif pour ce mois : l'ajustement explicite s'il existe, sinon la
// moyenne mensuelle réelle sur les mois déjà terminés — portage exact de
// suggestedCategoryBudget/currentMonthCategoryBudget de
// frontend/src/stores/purchases.js, pour que l'alerte email reflète
// fidèlement ce que l'écran Budgets afficherait.
func (s *Service) EffectiveBudgets(ctx context.Context, accountID uint64, monthKey string) (map[uint64]float64, error) {
	categories, err := s.categoryRepo.List(ctx, accountID)
	if err != nil {
		return nil, err
	}

	overrides, err := s.repo.GetForAccount(ctx, accountID, monthKey)
	if err != nil {
		return nil, err
	}

	purchases, err := s.purchaseRepo.List(ctx, accountID)
	if err != nil {
		return nil, err
	}

	accountIncomes, err := s.incomeRepo.List(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Nombre de mois déjà terminés (achats OU revenus, toutes catégories) —
	// dénominateur de la moyenne mensuelle réelle. Le mois en cours est
	// exclu car incomplet.
	pastMonths := make(map[string]bool)
	for _, p := range purchases {
		key := p.PurchaseDate.Format("2006-01")
		if key < monthKey {
			pastMonths[key] = true
		}
	}
	for _, i := range accountIncomes {
		key := i.IncomeDate.Format("2006-01")
		if key < monthKey {
			pastMonths[key] = true
		}
	}
	pastMonthCount := len(pastMonths)

	result := make(map[uint64]float64, len(categories))

	for _, cat := range categories {
		if amount, ok := overrides[cat.ID]; ok {
			result[cat.ID] = amount
			continue
		}

		if pastMonthCount == 0 {
			result[cat.ID] = 0
			continue
		}

		var total float64

		if cat.Type == category.TypeRevenu {
			for _, i := range accountIncomes {
				if i.Category == cat.Name && i.IncomeDate.Format("2006-01") < monthKey {
					total += i.Amount
				}
			}
		} else {
			for _, p := range purchases {
				if p.CategoryID == cat.ID && p.PurchaseDate.Format("2006-01") < monthKey {
					total += p.Amount
				}
			}
		}

		result[cat.ID] = total / float64(pastMonthCount)
	}

	return result, nil
}
