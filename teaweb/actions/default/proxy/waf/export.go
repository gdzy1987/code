package waf

import (
	"github.com/TeaWeb/code/teaconfigs"
	"github.com/TeaWeb/code/teawaf/rules"
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"strings"
)

type ExportAction actions.Action

// 导出
func (this *ExportAction) RunGet(params struct {
	WafId string

	Export   bool
	GroupIds string
}) {
	waf := teaconfigs.SharedWAFList().FindWAF(params.WafId)
	if waf == nil {
		this.Fail("找不到WAF")
	}

	// 导出
	if params.Export {
		waf1 := waf.Copy()
		waf1.RuleGroups = []*rules.RuleGroup{}
		if len(params.GroupIds) > 0 {
			groupIds := strings.Split(params.GroupIds, ",")
			for _, groupId := range groupIds {
				group := waf.FindRuleGroup(groupId)
				if group == nil {
					continue
				}
				waf1.AddRuleGroup(group)
			}
		}

		data, err := yaml.Marshal(waf1)
		if err != nil {
			this.WriteString(err.Error())
			return
		}

		filename := "waf." + waf1.Id + ".conf"
		this.ResponseWriter.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

		this.Write(data)

		return
	}

	this.Data["config"] = maps.Map{
		"id":        waf.Id,
		"name":      waf.Name,
		"countSets": waf.CountRuleSets(),
	}

	this.Data["groups"] = lists.Map(waf.RuleGroups, func(k int, v interface{}) interface{} {
		group := v.(*rules.RuleGroup)
		return maps.Map{
			"id":        group.Id,
			"name":      group.Name,
			"countSets": len(group.RuleSets),
			"on":        group.On,
		}
	})

	this.Show()
}
