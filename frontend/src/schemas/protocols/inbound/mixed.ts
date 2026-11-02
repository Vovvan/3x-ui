import { z } from 'zod';

export const MixedAuthSchema = z.enum(['password', 'noauth']);
export type MixedAuth = z.infer<typeof MixedAuthSchema>;

// SOCKS/HTTP combined inbound. When auth==='noauth' the `accounts` field is
// omitted from the wire payload (the panel writes `undefined`), so we accept
// either an array or absence here.
export const MixedAccountSchema = z.object({
  user: z.string().min(1),
  pass: z.string().min(1),
});
export type MixedAccount = z.infer<typeof MixedAccountSchema>;

// The modified core registers these as SOCKS users keyed by email; `password`
// is the panel client's credential and `level` defaults to the core's level 0.
export const MixedClientSchema = z.object({
  email: z.string().min(1),
  password: z.string().default(''),
  level: z.number().int().min(0).default(0),
});
export type MixedClient = z.infer<typeof MixedClientSchema>;

export const MixedInboundSettingsSchema = z.object({
  auth: MixedAuthSchema.default('password'),
  accounts: z.array(MixedAccountSchema).optional(),
  clients: z.array(MixedClientSchema).optional(),
  udp: z.boolean().default(false),
  ip: z.string().default('127.0.0.1'),
});
export type MixedInboundSettings = z.infer<typeof MixedInboundSettingsSchema>;
