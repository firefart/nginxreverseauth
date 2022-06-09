package main

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

type dnsClient struct {
	mu       sync.Mutex
	cache    map[string]dnsCacheEntry
	resolver *net.Resolver
	timeout  time.Duration
}

type dnsCacheEntry struct {
	names     []string
	timestamp time.Time
}

func newDNSClient(timeout time.Duration) *dnsClient {
	var r *net.Resolver

	return &dnsClient{
		cache:    make(map[string]dnsCacheEntry),
		resolver: r,
		timeout:  timeout,
	}
}

func (d *dnsClient) getCacheEntry(ip string) []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	val, ok := d.cache[ip]
	if !ok {
		return nil
	}

	if val.timestamp.Before(time.Now().Add(-1 * time.Hour)) {
		delete(d.cache, ip)
		return nil
	}

	return val.names
}

func (d *dnsClient) setCacheEntry(ip string, values []string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache[ip] = dnsCacheEntry{
		names:     values,
		timestamp: time.Now(),
	}
}

func (d *dnsClient) reverseLookup(ctx context.Context, ip string) ([]string, error) {
	cached := d.getCacheEntry(ip)
	if cached != nil {
		return cached, nil
	}

	ctx2, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	addr, err := d.resolver.LookupAddr(ctx2, ip)
	if err != nil {
		return nil, err
	}

	addrsCleaned := make([]string, len(addr))
	for i := range addr {
		addrsCleaned[i] = strings.TrimRight(addr[i], ".")
	}

	d.setCacheEntry(ip, addrsCleaned)

	return addrsCleaned, nil
}

func (d *dnsClient) ipLookup(ctx context.Context, domain string) ([]string, error) {
	cached := d.getCacheEntry(domain)
	if cached != nil {
		return cached, nil
	}

	ctx2, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	addr, err := d.resolver.LookupHost(ctx2, domain)
	if err != nil {
		return nil, err
	}

	d.setCacheEntry(domain, addr)

	return addr, nil
}
