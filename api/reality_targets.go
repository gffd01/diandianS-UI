package api

import (
	"context"
	"strconv"
	"time"

	"github.com/alireza0/s-ui/network/realitytarget"
	"github.com/alireza0/s-ui/util/common"

	"github.com/gin-gonic/gin"
)

type realityTargetResponse struct {
	Source  string                 `json:"source"`
	Tested  int                    `json:"tested"`
	Results []realitytarget.Result `json:"results"`
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

	// 候选池随 S-UI 一起发布，不下载公网榜单或第三方候选接口。
	domains := realitytarget.SelectBatch(realitytarget.BuiltInDomains, batch, 20)
	results := realitytarget.Scan(ctx, domains, 3, 8)

	jsonObj(c, realityTargetResponse{
		Source:  "local",
		Tested:  len(domains),
		Results: results,
	}, nil)
}
