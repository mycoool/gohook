package syncnode

import (
	"container/list"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type hashCacheValue struct {
	key    string
	hashes []string
}

type blockHashCache struct {
	mu         sync.Mutex
	maxEntries int
	ll         *list.List
	m          map[string]*list.Element
}

func newBlockHashCache() *blockHashCache {
	maxEntries := 2048
	if raw := strings.TrimSpace(os.Getenv("SYNC_HASHCACHE_ENTRIES")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			maxEntries = v
		}
	}
	return &blockHashCache{
		maxEntries: maxEntries,
		ll:         list.New(),
		m:          make(map[string]*list.Element),
	}
}

func (c *blockHashCache) Get(key string) ([]string, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.m[key]; ok {
		c.ll.MoveToFront(ele)
		v := ele.Value.(*hashCacheValue)
		return v.hashes, true
	}
	return nil, false
}

func (c *blockHashCache) Put(key string, hashes []string) {
	if c == nil || key == "" || len(hashes) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.m[key]; ok {
		c.ll.MoveToFront(ele)
		ele.Value.(*hashCacheValue).hashes = hashes
		return
	}
	ele := c.ll.PushFront(&hashCacheValue{key: key, hashes: hashes})
	c.m[key] = ele
	for c.ll.Len() > c.maxEntries {
		back := c.ll.Back()
		if back == nil {
			break
		}
		v := back.Value.(*hashCacheValue)
		delete(c.m, v.key)
		c.ll.Remove(back)
	}
}

var globalHashCache = newBlockHashCache()

func hashCacheKey(fullPath string, size int64, mod time.Time, blockSize int64) string {
	// Full path ensures uniqueness across projects. modtime nanoseconds avoids false hits.
	return strings.Join([]string{
		fullPath,
		strconv.FormatInt(size, 10),
		strconv.FormatInt(mod.UnixNano(), 10),
		strconv.FormatInt(blockSize, 10),
	}, "|")
}
