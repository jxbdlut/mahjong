package internal

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/jxbdlut/leaf/gate"
	"github.com/jxbdlut/leaf/log"
	"reflect"
	"server/game"
	"server/proto"
	"server/userdata"
	"strconv"
	"strings"
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
	seq := args[2].(uint32)

	log.Debug("login:%v, seq:%v", req, seq)
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strconv.FormatUint(req.Uid, 10)))
	cipherStr := md5Ctx.Sum(nil)
	password := hex.EncodeToString(cipherStr)
	if strings.Compare(password, req.Passwd) == 0 {
		a.SetUserData(&userdata.UserData{
			Uid: req.Uid,
		})
		need_recover := false
		if _, ok := game.MapUidPlayer[req.Uid]; ok {
			need_recover = true
		}
		player, ok := game.MapUidPlayer[req.Uid]
		if ok {
			player.SetAgent(a)
		}
		a.Replay(&proto.LoginRsp{
			ErrCode:     0,
			ErrMsg:      "login success",
			NeedRecover: need_recover,
		}, seq)
	} else {
		a.Replay(&proto.LoginRsp{
			ErrCode: -1,
			ErrMsg:  "account or password error!",
		}, seq)
	}

}
