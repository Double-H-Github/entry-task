// @Author: 2014BDuck
// @Date: 2021/8/3

package erpc_service

import (
	"context"
	"errors"
	"github.com/2014bduck/entry-task/global"
	"github.com/2014bduck/entry-task/internal/dao"
	"github.com/2014bduck/entry-task/pkg/rpc/erpc"
	"github.com/2014bduck/entry-task/pkg/upload"
	proto "github.com/2014bduck/entry-task/proto/erpc-proto"
	"os"
)

type UploadService struct {
	ctx   context.Context
	dao   *dao.Dao
	cache *dao.RedisCache
}

func NewUploadService(ctx context.Context) UploadService {
	svc := UploadService{ctx: ctx}
	svc.dao = dao.New(global.DBEngine)
	svc.cache = dao.NewCache(global.CacheClient)

	return svc
}

func (svc UploadService) RegisterUploadService(s *erpc.Server) {
	s.Register("UploadFile", svc.UploadFile, proto.UploadRequest{}, proto.UploadReply{})
}

func (svc UploadService) UploadFile(r proto.UploadRequest) (*proto.UploadReply, error) {
	fileName := upload.GetFileName(r.FileName) // MD5'd
	uploadSavePath := upload.GetSavePath()
	dst := uploadSavePath + "/" + fileName

	if upload.CheckSavePath(uploadSavePath) {
		if err := upload.CreateSavePath(dst, os.ModePerm); err != nil {
			return &proto.UploadReply{}, errors.New("svc.UploadFile: failed to create save directory")
		}
	}

	if upload.CheckPermission(uploadSavePath) {
		return &proto.UploadReply{}, errors.New("svc.UploadFile: insufficient file permissions")
	}
	if err := upload.SaveFileByte(&r.Content, dst); err != nil {
		return &proto.UploadReply{}, err
	}
	fileUrl := global.AppSetting.UploadServerUrl + "/" + fileName
	return &proto.UploadReply{FileUrl: fileUrl, FileName: fileName}, nil

}
