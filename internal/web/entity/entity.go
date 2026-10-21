package entity

import (
	"crypto/tls"
	"math"
	"net"
	"net/mail"
	"net/netip"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	"github.com/mhsanaei/3x-ui/v3/internal/util/cronspec"
)

type Msg struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Obj     any    `json:"obj"`
}

type AllSetting struct {
	WebListen             string `json:"webListen" form:"webListen"`
	WebDomain             string `json:"webDomain" form:"webDomain"`
	WebPort               int    `json:"webPort" form:"webPort" validate:"gte=1,lte=65535"`
	WebCertFile           string `json:"webCertFile" form:"webCertFile"`
	WebKeyFile            string `json:"webKeyFile" form:"webKeyFile"`
	WebBasePath           string `json:"webBasePath" form:"webBasePath"`
	SessionMaxAge         int    `json:"sessionMaxAge" form:"sessionMaxAge" validate:"gte=1,lte=525600"`
	TrustedProxyCIDRs     string `json:"trustedProxyCIDRs" form:"trustedProxyCIDRs"`
	RealityScanCandidates string `json:"realityScanCandidates" form:"realityScanCandidates"`
	IpLimitAllowlist      string `json:"ipLimitAllowlist" form:"ipLimitAllowlist"`
	PanelOutbound         string `json:"panelOutbound" form:"panelOutbound"`

	PageSize                   int    `json:"pageSize" form:"pageSize" validate:"gte=0,lte=1000"`
	ExpireDiff                 int    `json:"expireDiff" form:"expireDiff" validate:"gte=0"`
	TrafficDiff                int    `json:"trafficDiff" form:"trafficDiff" validate:"gte=0,lte=100"`
	RemarkTemplate             string `json:"remarkTemplate" form:"remarkTemplate"`
	SubShowIdentityOnAllLinks  bool   `json:"subShowIdentityOnAllLinks" form:"subShowIdentityOnAllLinks"`
	SubInfoNodeEnable          bool   `json:"subInfoNodeEnable" form:"subInfoNodeEnable"`
	SubHostSortMode            string `json:"subHostSortMode" form:"subHostSortMode" example:"inbound-first"`
	SubCalendarExpireInclusive bool   `json:"subCalendarExpireInclusive" form:"subCalendarExpireInclusive"`
	SubExpiredTemplate         string `json:"subExpiredTemplate" form:"subExpiredTemplate"`
	SubTrafficDepletedTemplate string `json:"subTrafficDepletedTemplate" form:"subTrafficDepletedTemplate"`
	Datepicker                 string `json:"datepicker" form:"datepicker"`

	SmtpEnable         bool   `json:"smtpEnable" form:"smtpEnable"`
	SmtpHost           string `json:"smtpHost" form:"smtpHost"`
	SmtpPort           int    `json:"smtpPort" form:"smtpPort" validate:"gte=1,lte=65535"`
	SmtpUsername       string `json:"smtpUsername" form:"smtpUsername"`
	SmtpPassword       string `json:"smtpPassword" form:"smtpPassword"`
	SmtpFrom           string `json:"smtpFrom" form:"smtpFrom"`
	SmtpFromName       string `json:"smtpFromName" form:"smtpFromName"`
	SmtpTo             string `json:"smtpTo" form:"smtpTo"`
	SmtpEncryptionType string `json:"smtpEncryptionType" form:"smtpEncryptionType"`
	SmtpEnabledEvents  string `json:"smtpEnabledEvents" form:"smtpEnabledEvents"`
	SmtpCpu            int    `json:"smtpCpu" form:"smtpCpu" validate:"gte=0,lte=100"`
	SmtpMemory         int    `json:"smtpMemory" form:"smtpMemory" validate:"gte=0,lte=100"`

	OutboundDownThreshold int `json:"outboundDownThreshold" form:"outboundDownThreshold" validate:"gte=1,lte=100"`
	NodeDownThreshold     int `json:"nodeDownThreshold" form:"nodeDownThreshold" validate:"gte=1,lte=100"`

	TimeLocation    string `json:"timeLocation" form:"timeLocation"`
	TwoFactorEnable bool   `json:"twoFactorEnable" form:"twoFactorEnable"`
	TwoFactorToken  string `json:"twoFactorToken" form:"twoFactorToken"`

	SubDomain                   string `json:"subDomain" form:"subDomain"`
	ExternalTrafficInformEnable bool   `json:"externalTrafficInformEnable" form:"externalTrafficInformEnable"`
	ExternalTrafficInformURI    string `json:"externalTrafficInformURI" form:"externalTrafficInformURI"`
	RestartXrayOnClientDisable  bool   `json:"restartXrayOnClientDisable" form:"restartXrayOnClientDisable"`

	LdapEnable             bool   `json:"ldapEnable" form:"ldapEnable"`
	LdapHost               string `json:"ldapHost" form:"ldapHost"`
	LdapPort               int    `json:"ldapPort" form:"ldapPort" validate:"gte=0,lte=65535"`
	LdapUseTLS             bool   `json:"ldapUseTLS" form:"ldapUseTLS"`
	LdapInsecureSkipVerify bool   `json:"ldapInsecureSkipVerify" form:"ldapInsecureSkipVerify"`
	LdapBindDN             string `json:"ldapBindDN" form:"ldapBindDN"`
	LdapPassword           string `json:"ldapPassword" form:"ldapPassword"`
	LdapBaseDN             string `json:"ldapBaseDN" form:"ldapBaseDN"`
	LdapUserFilter         string `json:"ldapUserFilter" form:"ldapUserFilter"`
	LdapUserAttr           string `json:"ldapUserAttr" form:"ldapUserAttr"`
	LdapVlessField         string `json:"ldapVlessField" form:"ldapVlessField"`
	LdapSyncCron           string `json:"ldapSyncCron" form:"ldapSyncCron"`
	LdapFlagField          string `json:"ldapFlagField" form:"ldapFlagField"`
	LdapTruthyValues       string `json:"ldapTruthyValues" form:"ldapTruthyValues"`
	LdapInvertFlag         bool   `json:"ldapInvertFlag" form:"ldapInvertFlag"`
	LdapInboundTags        string `json:"ldapInboundTags" form:"ldapInboundTags"`
	LdapAutoCreate         bool   `json:"ldapAutoCreate" form:"ldapAutoCreate"`
	LdapAutoDelete         bool   `json:"ldapAutoDelete" form:"ldapAutoDelete"`
	LdapDefaultTotalGB     int    `json:"ldapDefaultTotalGB" form:"ldapDefaultTotalGB" validate:"gte=0"`
	LdapDefaultExpiryDays  int    `json:"ldapDefaultExpiryDays" form:"ldapDefaultExpiryDays" validate:"gte=0"`
	LdapDefaultLimitIP     int    `json:"ldapDefaultLimitIP" form:"ldapDefaultLimitIP" validate:"gte=0"`

	WarpUpdateInterval int `json:"warpUpdateInterval" form:"warpUpdateInterval" validate:"gte=0"`
}

type AllSettingView struct {
	AllSetting

	HasTwoFactorToken bool `json:"hasTwoFactorToken"`
	HasLdapPassword   bool `json:"hasLdapPassword"`
	HasApiToken       bool `json:"hasApiToken"`
	HasWarpSecret     bool `json:"hasWarpSecret"`
	HasNordSecret     bool `json:"hasNordSecret"`
	HasSmtpPassword   bool `json:"hasSmtpPassword"`
}

func pathHasForbiddenChar(s string) bool {
	for _, r := range s {
		if r == '\\' || r == ' ' || r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// CheckNetipAddrOrPrefixList mirrors parseIpLimitAllowlist exactly: net and netip
// disagree (net accepts "/024", netip does not), so save and scan must share rules.
func CheckNetipAddrOrPrefixList(list, message string) error {
	for entry := range strings.SplitSeq(list, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, err := netip.ParseAddr(entry); err == nil {
			continue
		}
		if _, err := netip.ParsePrefix(entry); err != nil {
			return common.NewError(message, entry)
		}
	}
	return nil
}

// checkIPOrCIDRList rejects the first comma-separated entry that is neither a
// bare address nor a CIDR, naming it with the caller's message.
func checkIPOrCIDRList(list, message string) error {
	for entry := range strings.SplitSeq(list, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err != nil {
			return common.NewError(message, entry)
		}
	}
	return nil
}

func (s *AllSetting) CheckValid() error {
	if s.WebListen != "" {
		ip := net.ParseIP(s.WebListen)
		if ip == nil {
			return common.NewError("web listen is not valid ip:", s.WebListen)
		}
	}

	if s.WebPort <= 0 || s.WebPort > math.MaxUint16 {
		return common.NewError("web port is not a valid port:", s.WebPort)
	}

	if s.WebCertFile != "" || s.WebKeyFile != "" {
		_, err := tls.LoadX509KeyPair(s.WebCertFile, s.WebKeyFile)
		if err != nil {
			return common.NewErrorf("cert file <%v> or key file <%v> invalid: %v", s.WebCertFile, s.WebKeyFile, err)
		}
	}

	for _, p := range []struct {
		name  string
		value string
	}{
		{"web base path", s.WebBasePath},
	} {
		if pathHasForbiddenChar(p.value) {
			return common.NewError("URI path contains an invalid character:", p.name)
		}
	}

	if !strings.HasPrefix(s.WebBasePath, "/") {
		s.WebBasePath = "/" + s.WebBasePath
	}
	if !strings.HasSuffix(s.WebBasePath, "/") {
		s.WebBasePath += "/"
	}
	for _, sched := range []struct{ name, spec string }{
		{"ldapSyncCron", s.LdapSyncCron},
	} {
		if strings.TrimSpace(sched.spec) == "" {
			continue
		}
		if _, err := cronspec.Parse(sched.spec); err != nil {
			return common.NewError(sched.name+":", err)
		}
	}

	if err := checkIPOrCIDRList(s.TrustedProxyCIDRs, "trusted proxy CIDR is not valid:"); err != nil {
		return err
	}

	// Rejected here rather than skipped at scan time: a typo in an allowlist
	// entry silently leaves the address unprotected until a trusted network gets banned.
	if err := CheckNetipAddrOrPrefixList(s.IpLimitAllowlist, "IP limit allowlist entry is not valid:"); err != nil {
		return err
	}

	_, err := time.LoadLocation(s.TimeLocation)
	if err != nil {
		return common.NewError("time location not exist:", s.TimeLocation)
	}

	if s.SmtpFrom != "" {
		if _, err := mail.ParseAddress(s.SmtpFrom); err != nil {
			return common.NewError("SMTP from address is not valid:", s.SmtpFrom)
		}
	}

	return nil
}

// listenAddressesConflict reports whether two listen addresses on the same port
// would collide at bind time. A wildcard listen ("", "0.0.0.0", "::") overlaps
// every address, so it conflicts with anything on that port; two specific
// addresses conflict only when identical.
func listenAddressesConflict(a, b string) bool {
	if a == b {
		return true
	}
	return isWildcardListen(a) || isWildcardListen(b)
}

func isWildcardListen(listen string) bool {
	if listen == "" {
		return true
	}
	if ip := net.ParseIP(listen); ip != nil {
		return ip.IsUnspecified()
	}
	return false
}

type HostGroup struct {
	GroupId    string   `json:"groupId"`
	InboundIds []int    `json:"inboundIds" validate:"required,min=1"`
	Hosts      []string `json:"hosts" validate:"omitempty"`

	SortOrder              int      `json:"sortOrder"`
	Remark                 string   `json:"remark" validate:"required,max=256"`
	ServerDescription      string   `json:"serverDescription" validate:"omitempty,max=64"`
	IsDisabled             bool     `json:"isDisabled"`
	IsHidden               bool     `json:"isHidden"`
	Tags                   []string `json:"tags"`
	Port                   int      `json:"port" validate:"gte=0,lte=65535"`
	Security               string   `json:"security" validate:"omitempty,oneof=same tls none reality"`
	Sni                    string   `json:"sni"`
	Snis                   []string `json:"snis"`
	SniSharedRandom        bool     `json:"sniSharedRandom"`
	HostHeader             string   `json:"hostHeader"`
	Path                   string   `json:"path"`
	Alpn                   []string `json:"alpn"`
	Fingerprint            string   `json:"fingerprint"`
	CipherSuites           string   `json:"cipherSuites"`
	OverrideSniFromAddress bool     `json:"overrideSniFromAddress"`
	KeepSniBlank           bool     `json:"keepSniBlank"`
	PinnedPeerCertSha256   []string `json:"pinnedPeerCertSha256"`
	VerifyPeerCertByName   string   `json:"verifyPeerCertByName"`
	AllowInsecure          bool     `json:"allowInsecure"`
	EchConfigList          string   `json:"echConfigList"`
	MuxParams              string   `json:"muxParams"`
	SockoptParams          string   `json:"sockoptParams"`
	FinalMask              string   `json:"finalMask"`
	VlessRoute             string   `json:"vlessRoute"`
	ExcludeFromSubTypes    []string `json:"excludeFromSubTypes"`
	NodeGuids              []string `json:"nodeGuids"`
	MihomoIpVersion        string   `json:"mihomoIpVersion" validate:"omitempty,oneof=dual ipv4 ipv6 ipv4-prefer ipv6-prefer"`
	MihomoX25519           bool     `json:"mihomoX25519"`
	ShuffleHost            bool     `json:"shuffleHost"`
	RandomCount            int      `json:"randomCount" validate:"omitempty,min=1,max=100"`
}
