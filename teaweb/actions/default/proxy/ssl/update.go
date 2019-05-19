package ssl

import (
	"fmt"
	"github.com/TeaWeb/code/teaconfigs"
	"github.com/TeaWeb/code/teaweb/actions/default/proxy/proxyutils"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/files"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/utils/string"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type UpdateAction actions.Action

// 修改
func (this *UpdateAction) Run(params struct {
	ServerId string
}) {
	server := teaconfigs.NewServerConfigFromId(params.ServerId)
	if server == nil {
		this.Fail("找不到Server")
	}

	this.Data["certs"] = []maps.Map{}
	if server.SSL != nil {
		server.SSL.Validate()
		if len(server.SSL.Certs) > 0 {
			certs := []maps.Map{}
			for _, cert := range server.SSL.Certs {
				certs = append(certs, maps.Map{
					"id":          cert.Id,
					"on":          cert.On,
					"certFile":    cert.CertFile,
					"keyFile":     cert.KeyFile,
					"description": cert.Description,
				})
			}
			this.Data["certs"] = certs
		}
	}

	this.Data["selectedTab"] = "https"
	this.Data["server"] = server
	this.Data["versions"] = teaconfigs.AllTlsVersions

	this.Data["minVersion"] = "TLS 1.0"
	if server.SSL != nil && len(server.SSL.MinVersion) > 0 {
		this.Data["minVersion"] = server.SSL.MinVersion
	}

	// 加密算法套件
	this.Data["cipherSuites"] = teaconfigs.AllTLSCipherSuites

	this.Show()
}

// 提交保存
func (this *UpdateAction) RunPost(params struct {
	ServerId         string
	HttpsOn          bool
	Listen           []string
	CertIds          []string
	CertDescriptions []string
	MinVersion       string
	CipherSuitesOn   bool
	CipherSuites     []string
}) {
	server := teaconfigs.NewServerConfigFromId(params.ServerId)
	if server == nil {
		this.Fail("找不到Server")
	}

	if server.SSL == nil {
		server.SSL = teaconfigs.NewSSLConfig()
	}
	server.SSL.On = params.HttpsOn
	server.SSL.Listen = params.Listen

	if lists.ContainsString(teaconfigs.AllTlsVersions, params.MinVersion) {
		server.SSL.MinVersion = params.MinVersion
	}

	server.SSL.CipherSuites = []string{}
	if params.CipherSuitesOn {
		for _, cipherSuite := range params.CipherSuites {
			if lists.ContainsString(teaconfigs.AllTLSCipherSuites, cipherSuite) {
				server.SSL.CipherSuites = append(server.SSL.CipherSuites, cipherSuite)
			}
		}
	}

	fileBytes := map[string][]byte{} // field => []byte
	fileExts := map[string]string{}  // field => .ext
	if this.Request.MultipartForm != nil {
		for field, headers := range this.Request.MultipartForm.File {
			for _, header := range headers {
				fp, err := header.Open()
				if err != nil {
					continue
				}
				data, err := ioutil.ReadAll(fp)
				if err != nil {
					fp.Close()
					continue
				}
				fileBytes[field] = data
				fileExts[field] = strings.ToLower(filepath.Ext(header.Filename))
				fp.Close()

				break
			}
		}
	}

	// 证书
	certs := []*teaconfigs.SSLCertConfig{}
	for index, description := range params.CertDescriptions {
		if index >= len(params.CertIds) {
			continue
		}

		cert := teaconfigs.NewSSLCertConfig("", "")
		cert.Description = description

		// 兼容以前的版本（v0.1.4）
		if params.CertIds[index] == "old_version_cert" {
			cert.CertFile = server.SSL.Certificate
			cert.KeyFile = server.SSL.CertificateKey
		} else {
			// 保留先前上传的文件
			oldCert := server.SSL.FindCert(params.CertIds[index])
			if oldCert != nil {
				cert.CertFile = oldCert.CertFile
				cert.KeyFile = oldCert.KeyFile
			}
		}

		{
			field := fmt.Sprintf("certFiles%d", index)
			data, ok := fileBytes[field]
			if ok {
				filename := "ssl." + stringutil.Rand(16) + fileExts[field]
				configFile := files.NewFile(Tea.ConfigFile(filename))
				err := configFile.Write(data)
				if err != nil {
					this.Fail(err.Error())
				}
				cert.CertFile = filename
			}
		}

		{
			field := fmt.Sprintf("keyFiles%d", index)
			data, ok := fileBytes[field]
			if ok {
				filename := "ssl." + stringutil.Rand(16) + fileExts[field]
				configFile := files.NewFile(Tea.ConfigFile(filename))
				err := configFile.Write(data)
				if err != nil {
					this.Fail(err.Error())
				}
				cert.KeyFile = filename
			}
		}

		certs = append(certs, cert)
	}
	server.SSL.Certs = certs

	// 清除以前的版本（v0.1.4）
	server.SSL.Certificate = ""
	server.SSL.CertificateKey = ""

	err := server.Save()
	if err != nil {
		this.Fail("保存失败：" + err.Error())
	}

	proxyutils.NotifyChange()

	this.Success()
}
