package remote

import (
	"fmt"
	"sync"

	"github.com/ds-cli/ds-cli/internal/config"
)

type Pool struct {
	mu      sync.Mutex
	clients map[string]*Client
	cfg     *config.Config
}

func NewPool(cfg *config.Config) *Pool {
	return &Pool{clients: map[string]*Client{}, cfg: cfg}
}

func (p *Pool) Get(hostName string) (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.clients[hostName]; ok {
		return c, nil
	}
	h, ok := p.cfg.HostByName(hostName)
	if !ok {
		return nil, fmt.Errorf("unknown host: %s", hostName)
	}
	c, err := Dial(Config{
		Host:       h.Address,
		Port:       p.cfg.SSH.Port,
		User:       p.cfg.SSH.User,
		PrivateKey: p.cfg.SSH.PrivateKey,
	})
	if err != nil {
		return nil, err
	}
	p.clients[hostName] = c
	return c, nil
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.clients {
		_ = c.Close()
	}
	p.clients = map[string]*Client{}
}
