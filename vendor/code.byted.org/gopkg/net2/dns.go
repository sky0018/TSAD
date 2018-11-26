package net2

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type dnsCacheItem struct {
	utime time.Time
	addrs []string
}

var (
	resolver   = net.Resolver{PreferGo: true}
	dnsTimeout = 50 * time.Millisecond

	inmemoryCacheTimeout = time.Minute
	tmpfsCacheTimeout    = 12 * time.Hour

	dnsMu    sync.RWMutex
	dnscache = make(map[string]dnsCacheItem)
)

func LookupIPAddr(name string) ([]string, error) {
	dnsMu.RLock()
	ci := dnscache[name]
	dnsMu.RUnlock()

	if time.Since(ci.utime) < inmemoryCacheTimeout {
		return ci.addrs, nil
	}

	fi, err := os.Stat(cacheIPAddrFilename(name))
	if os.IsNotExist(err) {
		return lookupIPAddr(name)
	}
	if time.Since(fi.ModTime()) > tmpfsCacheTimeout {
		ret, err := lookupIPAddr(name) // cache timeout
		if err != nil {
			log.Println("[net2] LookupIPAddr: err", err)
		}
		if len(ret) > 0 {
			return ret, nil
		}
	}
	b, err := ioutil.ReadFile(cacheIPAddrFilename(name))
	if err != nil { // failback use lookupIPAddr
		log.Println("[net2] LookupIPAddr: Readfile err: %s", err)
		return lookupIPAddr(name)
	}
	ss := strings.TrimSpace(string(b))
	ret := strings.Split(ss, ";")
	dnsMu.Lock()
	dnscache[name] = dnsCacheItem{utime: time.Now(), addrs: ret}
	dnsMu.Unlock()
	return ret, nil
}

func cacheIPAddrFilename(name string) string {
	return filepath.Join(os.TempDir(), name+".cache")
}

func lookupIPAddr(name string) ([]string, error) {
	ctx, _ := context.WithDeadline(context.TODO(), time.Now().Add(dnsTimeout))
	ips, err := resolver.LookupIPAddr(ctx, name)
	ret := make([]string, len(ips))
	for i, ip := range ips {
		ret[i] = ip.String()
	}

	dnsMu.Lock()
	if len(ret) > 0 {
		dnscache[name] = dnsCacheItem{utime: time.Now(), addrs: ret}
	} else if ci, ok := dnscache[name]; ok {
		ci.utime = time.Now()
		dnscache[name] = ci
	}
	dnsMu.Unlock()

	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return ret, nil
	}
	// update cache file
	f, err := ioutil.TempFile("", name)
	if err != nil {
		log.Println("[net2] LookupIPAddr: open temp file err:", err)
		return ret, nil
	}
	_, err = f.Write([]byte(strings.Join(ret, ";")))
	f.Close()
	if err != nil {
		log.Println("[net2] LookupIPAddr: write temp file err:", err)
		return ret, nil
	}
	os.Rename(f.Name(), cacheIPAddrFilename(name)) // atomic
	return ret, nil
}
