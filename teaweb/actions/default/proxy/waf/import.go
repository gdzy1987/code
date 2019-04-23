package waf

import (
	"github.com/TeaWeb/code/teaconfigs"
	"github.com/TeaWeb/code/teawaf"
	"github.com/TeaWeb/code/teawaf/rules"
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/utils/string"
)

type ImportAction actions.Action

// 导入
func (this *ImportAction) RunGet(params struct {
	WafId string
}) {
	waf := teaconfigs.SharedWAFList().FindWAF(params.WafId)
	if waf == nil {
		this.Fail("找不到WAF")
	}

	this.Data["config"] = maps.Map{
		"id":        waf.Id,
		"name":      waf.Name,
		"countSets": waf.CountRuleSets(),
	}

	this.Show()
}

// 提交导入
func (this *ImportAction) RunPost(params struct {
	// step1
	File *actions.File

	// step2
	WafId    string
	GroupIds []string
	Data     string
	Step     string

	Must *actions.Must
}) {
	if params.Step == "file" {
		if params.File == nil {
			this.Fail("请上传要导入的规则集文件")
		}

		data, err := params.File.Read()
		if err != nil {
			this.Fail("文件读取失败：" + err.Error())
		}

		waf := &teawaf.WAF{}
		err = yaml.Unmarshal(data, waf)
		if err != nil {
			this.Fail("文件内容分析失败：" + err.Error())
		}

		this.Data["data"] = string(data)
		this.Data["groups"] = lists.Map(waf.RuleGroups, func(k int, v interface{}) interface{} {
			group := v.(*rules.RuleGroup)
			return maps.Map{
				"id":        group.Id,
				"name":      group.Name,
				"countSets": len(group.RuleSets),
			}
		})

		this.Success()
	} else if params.Step == "groups" { // 提交分组信息
		waf := &teawaf.WAF{}
		err := yaml.Unmarshal([]byte(params.Data), waf)
		if err != nil {
			this.Fail("文件内容分析失败：" + err.Error())
		}

		if len(params.GroupIds) == 0 {
			this.Fail("请选择要导入的规则分组")
		}

		wafList := teaconfigs.SharedWAFList()
		currentWAF := wafList.FindWAF(params.WafId)
		if currentWAF == nil {
			this.Fail("找不到当前的WAF")
		}

		countGroups := 0
		countSets := 0
		for _, groupId := range params.GroupIds {
			group := waf.FindRuleGroup(groupId)
			if group == nil {
				continue
			}
			countGroups ++
			countSets += len(group.RuleSets)
			group.Id = stringutil.Rand(16) // 重新生成ID，避免和现有的ID冲突
			currentWAF.AddRuleGroup(group)
		}

		err = wafList.SaveWAF(currentWAF)
		if err != nil {
			this.Fail("保存失败：" + err.Error())
		}

		this.Data["countGroups"] = countGroups
		this.Data["countSets"] = countSets

		this.Success()
	}
}
