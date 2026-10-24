import { createDefaultInboundSettings } from '@/lib/xray/inbound-defaults';
import { coerceInboundJsonField, type DBInbound } from '@/models/dbinbound';

/*
 * Payload for POST /panel/api/inbounds/add reproducing `dbInbound` as a
 * staged copy: fresh port, empty client list (emails are unique panel-wide),
 * disabled, no tag (the backend regenerates one), and cleared listen.
 */
export function buildClonePayload(dbInbound: DBInbound, port: number) {
  let clonedSettings: string;
  try {
    const raw = { ...coerceInboundJsonField(dbInbound.settings) };
    raw.clients = [];
    clonedSettings = JSON.stringify(raw);
  } catch {
    const fallback = createDefaultInboundSettings(dbInbound.protocol);
    clonedSettings = fallback ? JSON.stringify(fallback, null, 2) : '{}';
  }
  const streamSettingsString =
    typeof dbInbound.streamSettings === 'string'
      ? dbInbound.streamSettings
      : JSON.stringify(dbInbound.streamSettings ?? {});
  const sniffingString =
    typeof dbInbound.sniffing === 'string'
      ? dbInbound.sniffing
      : JSON.stringify(dbInbound.sniffing ?? {});
  return {
    up: 0,
    down: 0,
    total: 0,
    remark: `${dbInbound.remark} (clone)`,
    enable: false,
    expiryTime: 0,
    listen: '',
    port,
    protocol: dbInbound.protocol,
    settings: clonedSettings,
    streamSettings: streamSettingsString,
    sniffing: sniffingString,
    shareAddrStrategy: dbInbound.shareAddrStrategy,
    shareAddr: dbInbound.shareAddr,
  };
}
