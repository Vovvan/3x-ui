import { describe, expect, it } from 'vitest';

import { buildClonePayload } from '@/lib/xray/inbound-clone';
import { DBInbound } from '@/models/dbinbound';

function sourceInbound() {
  return new DBInbound({
    id: 7,
    port: 443,
    listen: '0.0.0.0',
    protocol: 'vless',
    remark: 'edge',
    enable: true,
    settings: JSON.stringify({
      clients: [{ id: 'uuid-1', email: 'a@test', flow: 'xtls-rprx-vision' }],
      decryption: 'none',
    }),
    streamSettings: {
      network: 'tcp',
      security: 'reality',
      realitySettings: { dest: 'www.lovelive-anime.jp:443' },
    },
    sniffing: { enabled: true },
    shareAddrStrategy: 'listen',
    shareAddr: '',
  });
}

describe('buildClonePayload', () => {
  it('stages the clone disabled with cleared clients, fresh port, and no tag', () => {
    const payload = buildClonePayload(sourceInbound(), 23456);
    expect(payload.enable).toBe(false);
    expect(payload.port).toBe(23456);
    expect(payload.listen).toBe('');
    expect(payload).not.toHaveProperty('tag');
    expect(payload.remark).toBe('edge (clone)');

    const settings = JSON.parse(payload.settings);
    // Clients are dropped (emails are unique panel-wide) while the rest of the
    // settings survive verbatim.
    expect(settings.clients).toEqual([]);
    expect(settings.decryption).toBe('none');
  });

  it('stringifies object-shaped streamSettings and sniffing from hydrated rows', () => {
    const payload = buildClonePayload(sourceInbound(), 23456);
    expect(JSON.parse(payload.streamSettings)).toEqual({
      network: 'tcp',
      security: 'reality',
      realitySettings: { dest: 'www.lovelive-anime.jp:443' },
    });
    expect(JSON.parse(payload.sniffing)).toEqual({ enabled: true });
  });

  it('survives malformed settings JSON with an empty client list fallback', () => {
    const broken = sourceInbound();
    broken.settings = '{not json';
    const payload = buildClonePayload(broken, 23456);
    const settings = JSON.parse(payload.settings);
    expect(settings.clients ?? []).toEqual([]);
  });
});
