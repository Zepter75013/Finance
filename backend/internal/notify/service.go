package notify

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"finance/backend/internal/account"
	"finance/backend/internal/budget"
	"finance/backend/internal/category"
	"finance/backend/internal/mailer"
	"finance/backend/internal/purchase"
	"finance/backend/internal/recurring"
	"finance/backend/internal/user"
)

const upcomingWindowDays = 7

type Service struct {
	mailer        *mailer.Mailer
	userRepo      *user.Repository
	accountRepo   *account.Repository
	categoryRepo  *category.Repository
	purchaseRepo  *purchase.Repository
	budgetService *budget.Service
	recurringRepo *recurring.Repository
}

func NewService(
	mailerClient *mailer.Mailer,
	userRepo *user.Repository,
	accountRepo *account.Repository,
	categoryRepo *category.Repository,
	purchaseRepo *purchase.Repository,
	budgetService *budget.Service,
	recurringRepo *recurring.Repository,
) *Service {
	return &Service{
		mailer:        mailerClient,
		userRepo:      userRepo,
		accountRepo:   accountRepo,
		categoryRepo:  categoryRepo,
		purchaseRepo:  purchaseRepo,
		budgetService: budgetService,
		recurringRepo: recurringRepo,
	}
}

// RunDailyDigestIfDue envoie, au plus une fois par jour civil, un résumé par
// email à chaque utilisateur ayant activé cette préférence — budgets
// dépassés et échéances de récurrence, tous comptes visibles confondus.
// Les conditions toujours vraies sont réincluses chaque jour (pas de
// mécanisme anti-doublon par alerte) : la seule porte est "le résumé du jour
// est-il déjà parti".
func (s *Service) RunDailyDigestIfDue(ctx context.Context) error {
	if !s.mailer.Configured() {
		return nil
	}

	today := time.Now().Format("2006-01-02")
	if loadLastDigestDate() == today {
		return nil
	}

	monthKey := time.Now().UTC().Format("2006-01")

	users, err := s.userRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("chargement des utilisateurs: %w", err)
	}

	for _, u := range users {
		if !u.EmailAlertsEnabled {
			continue
		}

		if err := s.sendDigestToUser(ctx, u, monthKey); err != nil {
			log.Printf("résumé quotidien: échec de l'envoi pour %s: %v", u.Username, err)
		}
	}

	return saveLastDigestDate(today)
}

func (s *Service) sendDigestToUser(ctx context.Context, u user.User, monthKey string) error {
	accounts, err := s.accountRepo.List(ctx, u.ID)
	if err != nil {
		return err
	}

	var sections []string
	overBudgetCount := 0
	upcomingCount := 0

	for _, acc := range accounts {
		section, overBudget, upcoming, err := s.accountSection(ctx, acc, monthKey)
		if err != nil {
			log.Printf("résumé quotidien: échec du calcul pour le compte %s: %v", acc.Name, err)
			continue
		}

		if section == "" {
			continue
		}

		sections = append(sections, section)
		overBudgetCount += overBudget
		upcomingCount += upcoming
	}

	if len(sections) == 0 {
		return nil
	}

	subject := fmt.Sprintf("Résumé quotidien : %d budget(s) dépassé(s), %d échéance(s)", overBudgetCount, upcomingCount)
	body := "Résumé quotidien - Finance UI\n\n" + strings.Join(sections, "\n\n")

	return s.mailer.Send(u.Username, subject, body)
}

func (s *Service) accountSection(ctx context.Context, acc account.Account, monthKey string) (section string, overBudgetCount int, upcomingCount int, err error) {
	var lines []string

	budgetLines, overBudgetCount, err := s.budgetLines(ctx, acc.ID, monthKey)
	if err != nil {
		return "", 0, 0, err
	}
	if len(budgetLines) > 0 {
		lines = append(lines, "Budgets dépassés :")
		lines = append(lines, budgetLines...)
	}

	recurringLines, upcomingCount, err := s.recurringLines(ctx, acc.ID)
	if err != nil {
		return "", 0, 0, err
	}
	if len(recurringLines) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Échéances à venir :")
		lines = append(lines, recurringLines...)
	}

	if len(lines) == 0 {
		return "", 0, 0, nil
	}

	header := fmt.Sprintf("Compte \"%s\"\n%s", acc.Name, strings.Repeat("-", len(acc.Name)+8))

	return header + "\n" + strings.Join(lines, "\n"), overBudgetCount, upcomingCount, nil
}

func (s *Service) budgetLines(ctx context.Context, accountID uint64, monthKey string) ([]string, int, error) {
	categories, err := s.categoryRepo.List(ctx, accountID)
	if err != nil {
		return nil, 0, err
	}

	effective, err := s.budgetService.EffectiveBudgets(ctx, accountID, monthKey)
	if err != nil {
		return nil, 0, err
	}

	purchases, err := s.purchaseRepo.List(ctx, accountID)
	if err != nil {
		return nil, 0, err
	}

	spent := make(map[uint64]float64)
	for _, p := range purchases {
		if p.PurchaseDate.UTC().Format("2006-01") == monthKey {
			spent[p.CategoryID] += p.Amount
		}
	}

	var lines []string

	for _, cat := range categories {
		if cat.Type == category.TypeRevenu {
			continue
		}

		budgetAmount := effective[cat.ID]
		spentAmount := spent[cat.ID]

		if spentAmount > budgetAmount {
			lines = append(lines, fmt.Sprintf(
				"  - %s : %.2f € dépensés / %.2f € (dépassement de %.2f €)",
				cat.Name, spentAmount, budgetAmount, spentAmount-budgetAmount,
			))
		}
	}

	return lines, len(lines), nil
}

func (s *Service) recurringLines(ctx context.Context, accountID uint64) ([]string, int, error) {
	items, err := s.recurringRepo.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, 0, err
	}

	var lines []string

	for _, item := range items {
		if !item.IsActive {
			continue
		}

		days := daysUntil(item.NextRunDate)
		if days > upcomingWindowDays {
			continue
		}

		label := ""
		if item.Type == recurring.TypeAchat && item.Merchant != nil {
			label = *item.Merchant
		} else if item.Source != nil {
			label = *item.Source
		}

		var whenLabel string
		switch {
		case days < 0:
			whenLabel = fmt.Sprintf("en retard de %d jour(s)", -days)
		case days == 0:
			whenLabel = "aujourd'hui"
		case days == 1:
			whenLabel = "demain"
		default:
			whenLabel = fmt.Sprintf("dans %d jours", days)
		}

		lines = append(lines, fmt.Sprintf("  - %s (échéance %s, %.2f €)", label, whenLabel, item.Amount))
	}

	return lines, len(lines), nil
}
