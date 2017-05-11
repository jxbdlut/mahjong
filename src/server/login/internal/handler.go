package internal

import (
	"crypto/md5"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"

	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/proto"
	"server/userdata"
)

func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
	handleMsg(&proto.LoginReq{}, handleLogin)
}

func handleLogin(args []interface{}) {
	req := args[0].(*proto.LoginReq)
	a := args[1].(gate.Agent)

	log.Debug("login:%v", req.String())
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strconv.FormatUint(req.Uid, 10)))
	cipherStr := md5Ctx.Sum(nil)
	passwd := hex.EncodeToString(cipherStr)
	log.Debug("passwd:%v", passwd)
	if strings.Compare(passwd, req.Passwd) == 0 {
		a.SetUserData(&userdata.UserData{
			Uid:  req.Uid,
		})
		a.WriteMsg(&proto.LoginRsp{
			ErrCode: 0,
			ErrMsg:  "successed",
		})
	} else {
		a.WriteMsg(&proto.LoginRsp{
			ErrCode: -1,
			ErrMsg:  "account or passwd error!",
		})
	}

}
