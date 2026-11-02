package api

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alireza0/s-ui/network/realitytarget"
	"github.com/alireza0/s-ui/util/common"

	"github.com/gin-gonic/gin"
)

const realityTargetFeedURL = "https://raw.githubusercontent.com/gffd01/diandianS-UI/reality-targets/reality-targets.json"

type realityTargetResponse struct {
	Source      string                 `json:"source"`
	GeneratedAt string                 `json:"generated_at,omitempty"`
	Fallback    bool                   `json:"fallback"`
	Tested      int                    `json:"tested"`
	Results     []realitytarget.Result `json:"results"`
}

type realityFeedCacheValue struct {
	domains     []string
	generatedAt string
}

var realityFeedCache struct {
	sync.RWMutex
	value realityFeedCacheValue
}

// 限制同时进行的扫描，避免反复点击给小内存 VPS 带来过多连接。
var realityScanSlots = make(chan struct{}, 2)

func (a *ApiService) GetRealityTargets(c *gin.Context) {
	select {
	case realityScanSlots <- struct{}{}:
		defer func() { <-realityScanSlots }()
	default:
		jsonMsg(c, "", common.NewError("测速任务较多，请稍后再试"))
		return
	}

	batch, _ := strconv.ParseUint(c.DefaultQuery("batch", "0"), 10, 64)
	// 中央管理转发接口的总超时为 30 秒，这里预留响应传输时间。
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	domains, source, generatedAt, fallback := loadRealityCandidates(ctx)
	domains = realitytarget.SelectBatch(domains, batch, 20)
	results := realitytarget.Scan(ctx, domains, 3, 8)

	jsonObj(c, realityTargetResponse{
		Source:      source,
		GeneratedAt: generatedAt,
		Fallback:    fallback,
		Tested:      len(domains),
		Results:     results,
	}, nil)
}

func loadRealityCandidates(ctx context.Context) ([]string, string, string, bool) {
	feed, err := realitytarget.FetchFeed(ctx, realityTargetFeedURL)
	if err == nil && len(feed.Domains) > 0 {
		realityFeedCache.Lock()
		realityFeedCache.value = realityFeedCacheValue{
			domains:     append([]string(nil), feed.Domains...),
			generatedAt: feed.GeneratedAt,
		}
		realityFeedCache.Unlock()
		return feed.Domains, "public", feed.GeneratedAt, false
	}

	// 公网临时不可用时优先使用本进程最近一次成功取得的候选库。
	realityFeedCache.RLock()
	cached := realityFeedCache.value
	realityFeedCache.RUnlock()
	if len(cached.domains) > 0 {
		return append([]string(nil), cached.domains...), "cache", cached.generatedAt, true
	}
	return append([]string(nil), realitytarget.BuiltInDomains...), "builtin", "", true
}
