package trip

type Service struct {
	tripRepo    tripRepository
	historyRepo tripHistoryRepository
	txManager   txManager
}

func New(tripRepo tripRepository, historyRepo tripHistoryRepository, txManager txManager) *Service {
	return &Service{
		tripRepo:    tripRepo,
		historyRepo: historyRepo,
		txManager:   txManager,
	}
}
