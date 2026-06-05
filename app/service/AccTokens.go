package service

import (
	"chat2api/app/acc_token_pool"
	"chat2api/pkg/logx"
	"fmt"

	"github.com/gin-gonic/gin"
)

type AccTokensResp struct {
	Count       int `json:"count"`
	CanUseCount int `json:"can_use_count"`
}

func AccTokens(c *gin.Context) {
	resp := &AccTokensResp{
		Count:       acc_token_pool.GetAccAuthPoolInstance().Size(),
		CanUseCount: acc_token_pool.GetAccAuthPoolInstance().CanUseSize(),
	}
	logx.WithContext(c.Request.Context()).Info(fmt.Sprint("AccessTokenPool Tokens: ", resp.Count))
	c.JSON(200, resp)
}
