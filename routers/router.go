// @APIVersion 1.0.0
// @Title beego Test API
// @Description beego has a very cool tools to autogenerate documents for your API
// @Contact astaxie@gmail.com
// @TermsOfServiceUrl http://beego.me/
// @License Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0.html
package routers

import (
	"github.com/braior/kube-devops-api/controllers"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/v1",
		beego.NSRouter("/list", &controllers.ResourceController{}, "get:List"),
		beego.NSRouter("/get", &controllers.ResourceController{}, "get:Get"),
		beego.NSRouter("/create", &controllers.ResourceController{}, "post:Create"),
	)
	beego.AddNamespace(ns)
}
