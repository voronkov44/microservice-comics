package words

type Service interface {
	Norm(phrase string) ([]string, error)
}

type service struct{}

func NewService() Service {
	return &service{}
}
