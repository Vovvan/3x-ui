import { ObjectUtil } from '@/utils';

export class AllSetting {
  webListen = '';
  webDomain = '';
  webPort = 2053;
  webCertFile = '';
  webKeyFile = '';
  webBasePath = '/';
  sessionMaxAge = 360;
  trustedProxyCIDRs = '127.0.0.1/32,::1/128';
  realityScanCandidates =
    'www.cloudflare.com:443,www.microsoft.com:443,www.amazon.com:443,aws.amazon.com:443,www.samsung.com:443,www.nvidia.com:443,www.amd.com:443,www.intel.com:443,www.sony.com:443,dl.google.com:443';
  ipLimitAllowlist = '';
  panelOutbound = '';
  pageSize = 25;
  expireDiff = 0;
  trafficDiff = 0;
  remarkTemplate = '{{INBOUND}}-{{EMAIL}}|📊{{TRAFFIC_LEFT}}|⏳{{DAYS_LEFT}}D';
  subShowIdentityOnAllLinks = false;
  subInfoNodeEnable = false;
  subHostSortMode: SubHostSortMode = 'inbound-first';
  subCalendarExpireInclusive = false;
  subExpiredTemplate = '⛔ {{EMAIL}} | Expired: {{EXPIRE_DATE}}';
  subTrafficDepletedTemplate =
    '🚫 {{EMAIL}} | Traffic Depleted | {{TRAFFIC_USED}}/{{TRAFFIC_TOTAL}}';
  datepicker: 'gregorian' | 'jalalian' = 'gregorian';
  twoFactorEnable = false;
  twoFactorToken = '';
  xrayTemplateConfig = '';
  subDomain = '';
  externalTrafficInformEnable = false;
  externalTrafficInformURI = '';
  restartXrayOnClientDisable = true;
  timeLocation = 'Local';

  ldapEnable = false;
  ldapHost = '';
  ldapPort = 389;
  ldapUseTLS = false;
  ldapInsecureSkipVerify = false;
  ldapBindDN = '';
  ldapPassword = '';
  ldapBaseDN = '';
  ldapUserFilter = '(objectClass=person)';
  ldapUserAttr = 'mail';
  ldapVlessField = 'vless_enabled';
  ldapSyncCron = '@every 1m';
  ldapFlagField = '';
  ldapTruthyValues = 'true,1,yes,on';
  ldapInvertFlag = false;
  ldapInboundTags = '';
  ldapAutoCreate = false;
  ldapAutoDelete = false;
  ldapDefaultTotalGB = 0;
  ldapDefaultExpiryDays = 0;
  ldapDefaultLimitIP = 0;
  smtpEnable = false;
  smtpHost = '';
  smtpPort = 587;
  smtpUsername = '';
  smtpPassword = '';
  smtpFrom = '';
  smtpFromName = '';
  smtpTo = '';
  smtpEncryptionType = 'starttls';
  smtpEnabledEvents = 'login.attempt,cpu.high';
  smtpCpu = 80;
  smtpMemory = 80;
  outboundDownThreshold = 3;
  nodeDownThreshold = 1;
  hasTgBotToken = false;
  hasTwoFactorToken = false;
  hasLdapPassword = false;
  hasApiToken = false;
  hasWarpSecret = false;
  hasNordSecret = false;
  hasSmtpPassword = false;
  clearTgBotToken = false;
  clearLdapPassword = false;
  clearSmtpPassword = false;
  hasDiscordBotToken = false;
  clearDiscordBotToken = false;

  constructor(data?: unknown) {
    if (data != null) {
      ObjectUtil.cloneProps(this, data);
    }
    const threshold = Math.round(Number(this.outboundDownThreshold));
    this.outboundDownThreshold = Number.isFinite(threshold)
      ? Math.min(100, Math.max(1, threshold))
      : 3;
    const nodeThreshold = Math.round(Number(this.nodeDownThreshold));
    this.nodeDownThreshold = Number.isFinite(nodeThreshold)
      ? Math.min(100, Math.max(1, nodeThreshold))
      : 1;
  }

  equals(other: AllSetting): boolean {
    return ObjectUtil.equals(this, other);
  }
}
