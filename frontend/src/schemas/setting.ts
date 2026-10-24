import { z } from 'zod';

const port = z.number().int().min(1).max(65535);
const nonNegativeInt = z.number().int().min(0);
const absolutePath = z.string().regex(/^\//, 'pages.settings.validation.pathLeadingSlash');

export const AllSettingSchema = z
  .object({
    webListen: z.string().optional(),
    webDomain: z.string().optional(),
    webPort: port.optional(),
    webCertFile: z.string().optional(),
    webKeyFile: z.string().optional(),
    webBasePath: absolutePath.optional(),
    sessionMaxAge: z.number().int().min(1).max(525600).optional(),
    trustedProxyCIDRs: z.string().optional(),
    realityScanCandidates: z.string().optional(),
    ipLimitAllowlist: z.string().optional(),
    panelOutbound: z.string().optional(),
    pageSize: z.number().int().min(0).max(1000).optional(),
    expireDiff: nonNegativeInt.optional(),
    trafficDiff: nonNegativeInt.max(100).optional(),
    remarkTemplate: z.string().optional(),
    subShowIdentityOnAllLinks: z.boolean().optional(),
    subInfoNodeEnable: z.boolean().optional(),
    subHostSortMode: SubHostSortModeSchema.optional(),
    subCalendarExpireInclusive: z.boolean().optional(),
    subExpiredTemplate: z.string().optional(),
    subTrafficDepletedTemplate: z.string().optional(),
    datepicker: z.enum(['gregorian', 'jalalian']).optional(),
    tgBotEnable: z.boolean().optional(),
    tgBotToken: z.string().optional(),
    tgBotProxy: z.string().optional(),
    tgBotAPIServer: z.string().optional(),
    tgBotChatId: z.string().optional(),
    tgRunTime: z.string().optional(),
    tgBotBackup: z.boolean().optional(),
    tgCpu: z.number().int().min(0).max(100).optional(),
    outboundDownThreshold: z.number().int().min(1).max(100).optional(),
    nodeDownThreshold: z.number().int().min(1).max(100).optional(),
    tgLang: z.string().optional(),
    twoFactorEnable: z.boolean().optional(),
    twoFactorToken: z.string().optional(),
    xrayTemplateConfig: z.string().optional(),
    subDomain: z.string().optional(),
    externalTrafficInformEnable: z.boolean().optional(),
    externalTrafficInformURI: z.string().optional(),
    restartXrayOnClientDisable: z.boolean().optional(),
    timeLocation: z.string().optional(),
    ldapEnable: z.boolean().optional(),
    ldapHost: z.string().optional(),
    ldapPort: port.optional(),
    ldapUseTLS: z.boolean().optional(),
    ldapInsecureSkipVerify: z.boolean().optional(),
    ldapBindDN: z.string().optional(),
    ldapPassword: z.string().optional(),
    ldapBaseDN: z.string().optional(),
    ldapUserFilter: z.string().optional(),
    ldapUserAttr: z.string().optional(),
    ldapVlessField: z.string().optional(),
    ldapSyncCron: z.string().optional(),
    ldapFlagField: z.string().optional(),
    ldapTruthyValues: z.string().optional(),
    ldapInvertFlag: z.boolean().optional(),
    ldapInboundTags: z.string().optional(),
    ldapAutoCreate: z.boolean().optional(),
    ldapAutoDelete: z.boolean().optional(),
    ldapDefaultTotalGB: nonNegativeInt.optional(),
    ldapDefaultExpiryDays: nonNegativeInt.optional(),
    ldapDefaultLimitIP: nonNegativeInt.optional(),
    hasTgBotToken: z.boolean().optional(),
    hasTwoFactorToken: z.boolean().optional(),
    hasLdapPassword: z.boolean().optional(),
    hasApiToken: z.boolean().optional(),
    hasWarpSecret: z.boolean().optional(),
    hasNordSecret: z.boolean().optional(),
    hasSmtpPassword: z.boolean().optional(),
    hasDiscordBotToken: z.boolean().optional(),
    discordBotEnable: z.boolean().optional(),
    discordBotToken: z.string().optional(),
    discordChannelId: z.string().optional(),
    discordAdminIds: z.string().optional(),
    discordRunTime: z.string().optional(),
    discordBotBackup: z.boolean().optional(),
    discordCpu: z.number().int().min(0).max(100).optional(),
    discordMemory: z.number().int().min(0).max(100).optional(),
    discordLang: z.string().optional(),
    discordEnabledEvents: z.string().optional(),
  })
  .loose();

export type AllSettingInput = z.infer<typeof AllSettingSchema>;

// Existing installations can contain regex values saved before the backend
// enforced its 2,048-character limit. Accept those values when reading so the
// settings page can display and let users correct them, while keeping the
// stricter schema above for outgoing updates.
export const AllSettingResponseSchema = AllSettingSchema.extend({});

export const FactoryDefaultsSchema = z.record(z.string(), z.string());

export type FactoryDefaults = z.infer<typeof FactoryDefaultsSchema>;
