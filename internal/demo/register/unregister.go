package register

import (
	"Open_IM/pkg/common/config"
	"Open_IM/pkg/common/constant"
	"Open_IM/pkg/common/db/mysql_model/im_mysql_model"
	"Open_IM/pkg/common/log"
	"Open_IM/pkg/common/token_verify"
	"Open_IM/pkg/grpc-etcdv3/getcdv3"
	"Open_IM/pkg/utils"
	"context"
	"net/http"
	"strings"

	server_api_params "Open_IM/pkg/proto/sdk_ws"
	rpc "Open_IM/pkg/proto/user"

	"github.com/gin-gonic/gin"
)

type unregisterReq struct {
	OperationID string `json:"operationID" binding:"required"`
	UserID      string `json:"userID" binding:"required"`
	Acount      string `json:"account" binding:"required"`
}

type unregisterResp struct {
}

func Unregister(c *gin.Context) {
	req := &unregisterReq{}
	if err := c.BindJSON(&req); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "bind json failed", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"errCode": constant.FormattingError, "errMsg": err.Error()})
		return
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), req)
	ok, opUserID, errInfo := token_verify.GetUserIDFromToken(c.Request.Header.Get("token"), req.OperationID)
	if !ok {
		errMsg := req.OperationID + " " + "GetUserIDFromToken failed " + errInfo + " token:" + c.Request.Header.Get("token")
		log.NewError(req.OperationID, errMsg)
		c.JSON(http.StatusBadRequest, gin.H{"errCode": 400, "errMsg": errMsg})
		return
	}
	if !utils.IsContain(opUserID, config.Config.Manager.AppManagerUid) || req.UserID != opUserID {
		log.NewError(req.OperationID, *req, "no permission", opUserID)
		c.JSON(http.StatusBadRequest, gin.H{"errCode": 400, "errMsg": "no permission"})
		return
	}
	err := im_mysql_model.UnregisterUser(req.Acount, req.UserID)
	if err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), err.Error(), req.Acount, req.UserID)
		c.JSON(http.StatusOK, gin.H{"errCode": 500, "errMsg": err.Error()})
		return
	}
	etcdConn := getcdv3.GetDefaultConn(config.Config.Etcd.EtcdSchema, strings.Join(config.Config.Etcd.EtcdAddr, ","), config.Config.RpcRegisterName.OpenImUserName, req.OperationID)
	if etcdConn == nil {
		errMsg := req.OperationID + "getcdv3.GetDefaultConn == nil"
		log.NewError(req.OperationID, errMsg)
		c.JSON(http.StatusInternalServerError, gin.H{"errCode": 500, "errMsg": errMsg})
		return
	}
	client := rpc.NewUserClient(etcdConn)
	rpcReq := &rpc.UpdateUserInfoReq{UserInfo: &server_api_params.UserInfo{UserID: req.UserID, Nickname: config.Config.Demo.UserUnregisterName}, OpUserID: opUserID, OperationID: req.OperationID}
	rpcResp, err := client.UpdateUserInfo(context.Background(), rpcReq)
	if err != nil {
		log.NewError(req.OperationID, "UpdateUserInfo failed ", err.Error(), rpcReq.String())
		c.JSON(http.StatusInternalServerError, gin.H{"errCode": 500, "errMsg": "call rpc server failed"})
		return
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), rpcResp)
	c.JSON(http.StatusOK, gin.H{"errCode": rpcResp.CommonResp.ErrCode, "errMsg": rpcResp.CommonResp.ErrMsg})
}
