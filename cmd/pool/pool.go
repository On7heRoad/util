package pool

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	ErrPoolClosed = errors.New("Pool has closed!")
)

type Pool struct {
	factory   func() (io.Closer, error)
	resources chan io.Closer
	mtx       sync.Mutex
	closed    bool
}

func New(factory func() (io.Closer, error), size uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("invalid size for the resources pool")
	}
	resource := make(chan io.Closer, size)
	p := Pool{
		resources: resource,
		closed:    false,
	}
	for i := 0; i < int(size); i++ {
		go func() {
			resource, err := factory()
			if err != nil {
				fmt.Printf("Create resource failed,error:%s", err)
			}
			p.resources <- resource
		}()
	}
	return &p, nil
}

func (p *Pool) AcquireResource() (io.Closer, error) {
	select {
	case resource, ok := <-p.resources:
		if !ok {
			return nil, ErrPoolClosed
		}
		fmt.Println("get resource from pool")
		return resource, nil
	}
}

func (p *Pool) ReleaseResource(resource io.Closer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.closed {
		resource.Close()
		return
	}
	select {
	case p.resources <- resource: // 如果能够丢进去
		fmt.Println("release resource back to the pool")
	default:
		fmt.Println("release resource closed")
		resource.Close()
	}
}

func (p *Pool) Close() {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.resources)
	for resource := range p.resources {
		resource.Close()
	}
}
