package controllers

import (
	"fmt"
	"strings"

	"github.com/astaxie/beego"
)

type BaseController struct {
	beego.Controller
}

type ResourceController struct {
	BaseController
}

type MsgReturn struct {
	EntryType string
	Status    string
	Msg       interface{}
	Data      interface{}
}

type EnterType struct {
	get    string
	list   string
	delete string
	create string
}

func ErrNotFoundDatacenter(datacenter string) error {
	err := fmt.Errorf("the '%s' datacenter was not found", datacenter)
	return err
}

func (b *BaseController) Json(entryType, status string, message, data interface{}) {
	res := MsgReturn{
		EntryType: entryType,
		Status:    status,
		Msg:       message,
		Data:      data,
	}
	b.Data["json"] = res
	b.ServeJSON()
	//b.StopRun()
}

func (b *BaseController) EntryType(resourceType string) string {
	action := strings.Split(b.Ctx.Request.URL.Path, "/")
	return strings.ToUpper(action[len(action)-1] + "_" + resourceType)
}
