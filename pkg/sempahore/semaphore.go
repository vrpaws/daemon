package sempahore

type Semaphore struct {
	c chan struct{}
}

func New(size int) *Semaphore {
	return &Semaphore{make(chan struct{}, size)}
}

func (s *Semaphore) Acquire() {
	s.c <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.c
}
