// Package provider defines built-in mail server presets for common providers.
//
// Presets only encode non-sensitive connection details (hosts, ports, SSL).
// Authentication always uses the account email plus an authorization code /
// client-specific password supplied by the user.
package provider

import "sort"

// Server describes an IMAP or SMTP endpoint.
type Server struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	// SSL true means implicit TLS (e.g. port 993 for IMAP, 465 for SMTP).
	SSL bool `json:"ssl"`
}

// Preset is a built-in mail provider configuration.
type Preset struct {
	Name        string
	IMAP        Server
	SMTP        Server
	Description string
}

// Presets returns the built-in provider presets keyed by name.
func Presets() map[string]Preset {
	return map[string]Preset{
		"qq": {
			Name:        "qq",
			IMAP:        Server{Host: "imap.qq.com", Port: 993, SSL: true},
			SMTP:        Server{Host: "smtp.qq.com", Port: 465, SSL: true},
			Description: "QQ 邮箱 (imap.qq.com:993 / smtp.qq.com:465，使用授权码)",
		},
		"aliyun-qiye": {
			Name:        "aliyun-qiye",
			IMAP:        Server{Host: "imap.qiye.aliyun.com", Port: 993, SSL: true},
			SMTP:        Server{Host: "smtp.qiye.aliyun.com", Port: 465, SSL: true},
			Description: "阿里企业邮箱 (imap.qiye.aliyun.com:993 / smtp.qiye.aliyun.com:465，使用客户端专用密码)",
		},
		"aliyun-qiye-hk": {
			Name:        "aliyun-qiye-hk",
			IMAP:        Server{Host: "imaphk.qiye.aliyun.com", Port: 993, SSL: true},
			SMTP:        Server{Host: "smtphk.qiye.aliyun.com", Port: 465, SSL: true},
			Description: "阿里企业邮箱-香港 (imaphk/smtphk.qiye.aliyun.com)",
		},
	}
}

// Get returns the preset with the given name.
func Get(name string) (Preset, bool) {
	p, ok := Presets()[name]
	return p, ok
}

// Names returns the sorted list of preset names.
func Names() []string {
	m := Presets()
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
