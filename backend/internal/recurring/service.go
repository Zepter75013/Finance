package recurring

import (
	"context"
	"errors"
	"log"
	"time"

	"finance/backend/internal/incomes"
	"finance/backend/internal/purchase"
)

// ErrNotDue est renvoyée par ExecuteNow quand la prochaine occurrence d'un
// gabarit n'est pas encore arrivée — le rattrapage manuel n'a de sens que
// pour une échéance déjà passée.
var ErrNotDue = errors.New("recurring: next run date is not due yet")

// Service exécute les gabarits arrivés à échéance en créant la transaction
// réelle correspondante (achat ou revenu), puis avance leur prochaine date.
type Service struct {
	repo         *Repository
	purchaseRepo *purchase.Repository
	incomeRepo   *incomes.Repository
}

func NewService(repo *Repository, purchaseRepo *purchase.Repository, incomeRepo *incomes.Repository) *Service {
	return &Service{repo: repo, purchaseRepo: purchaseRepo, incomeRepo: incomeRepo}
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func uint64OrZero(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}

// execute crée la transaction réelle pour un gabarit dû, puis avance sa
// prochaine occurrence. Chaque échec est retourné à l'appelant, qui décide
// s'il continue avec les gabarits suivants — un gabarit en échec ne doit
// jamais bloquer les autres.
func (s *Service) execute(ctx context.Context, tx RecurringTransaction) error {
	dateStr := tx.NextRunDate.Format("2006-01-02")

	switch tx.Type {
	case TypeAchat:
		_, err := s.purchaseRepo.Create(ctx, purchase.CreatePurchaseInput{
			Merchant:      stringOrEmpty(tx.Merchant),
			PaymentMethod: stringOrEmpty(tx.PaymentMethod),
			CategoryID:    uint64OrZero(tx.CategoryID),
			AccountID:     tx.AccountID,
			Amount:        tx.Amount,
			PurchaseDate:  dateStr,
			Note:          stringOrEmpty(tx.Note),
			SubCategory:   tx.SubCategory,
		})
		if err != nil {
			return err
		}
	case TypeRevenu:
		_, err := s.incomeRepo.Create(ctx, incomes.CreateIncomeInput{
			AccountID:     tx.AccountID,
			Source:        stringOrEmpty(tx.Source),
			Amount:        tx.Amount,
			IncomeDate:    dateStr,
			Note:          stringOrEmpty(tx.Note),
			OperationType: stringOrEmpty(tx.OperationType),
			Category:      stringOrEmpty(tx.Category),
			SubCategory:   tx.SubCategory,
		})
		if err != nil {
			return err
		}
	}

	return s.repo.AdvanceAfterExecution(ctx, tx.ID, tx.NextRunDate, tx.DayOfMonth)
}

// ExecuteNow réalise manuellement UNE occurrence d'un gabarit en retard
// (échéance déjà passée). Permet de rattraper mois par mois plutôt que
// d'attendre le prochain passage horaire du planificateur — chaque appel ne
// crée qu'une seule transaction et avance d'un mois ; s'il reste plusieurs
// mois de retard, l'appelant doit rappeler cette méthode autant de fois que
// nécessaire (un par mois manqué, jamais un rattrapage groupé silencieux).
func (s *Service) ExecuteNow(ctx context.Context, id uint64) (RecurringTransaction, error) {
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return RecurringTransaction{}, err
	}

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	if tx.NextRunDate.After(todayDate) {
		return RecurringTransaction{}, ErrNotDue
	}

	if err := s.execute(ctx, tx); err != nil {
		return RecurringTransaction{}, err
	}

	return s.repo.GetByID(ctx, id)
}

// RunDue exécute tous les gabarits arrivés à échéance à la date donnée. Ne
// remonte jamais d'erreur fatale : chaque échec est logué individuellement,
// le traitement des gabarits suivants continue.
func (s *Service) RunDue(ctx context.Context, asOf time.Time) {
	due, err := s.repo.DueForExecution(ctx, asOf)
	if err != nil {
		log.Printf("recurring: échec de la recherche des gabarits dus: %v", err)
		return
	}

	for _, tx := range due {
		if err := s.execute(ctx, tx); err != nil {
			log.Printf("recurring: échec de l'exécution du gabarit %d: %v", tx.ID, err)
		}
	}
}
