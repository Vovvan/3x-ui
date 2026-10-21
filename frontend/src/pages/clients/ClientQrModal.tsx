import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Collapse, Modal, Spin, Tag } from 'antd';
import { HttpUtil } from '@/utils';
import { isPostQuantumLink } from '@/lib/xray/inbound-link';
import { LinkTags, linkMetaText, parseLinkParts } from '@/lib/xray/link-label';
import { QrPanel } from '@/pages/inbounds/qr';
import type { ClientRecord, InboundOption } from '@/hooks/useClients';
import { formatTunnelConfigMeta } from '@/lib/inbounds/label';
import {
  buildWireguardClientConfig,
  findWireguardInbounds,
  isWireguardClient,
} from './wireguardConfig';
import {
  buildAmneziaWGClientConfig,
  findAmneziaWGInbounds,
  isAmneziaWGClient,
} from './amneziawgConfig';
import { buildTuicClientConfig, findTuicInbound, isTuicClient } from './tuicConfig';
import { tunnelConfigEndpoints, tunnelEndpointLabel } from './tunnelEndpoints';
import type { HostRecord } from '@/schemas/api/host';

interface ClientQrSettings {
  publicHost?: string;
}

interface ClientQrModalProps {
  open: boolean;
  client: ClientRecord | null;
  inboundsById: Record<number, InboundOption>;
  tunnelAllowedIPs?: Record<number, string>;
  subSettings?: ClientQrSettings;
  hosts?: HostRecord[];
  onOpenChange: (open: boolean) => void;
}

const NO_HOSTS: HostRecord[] = [];

interface ApiMsg<T = unknown> {
  success?: boolean;
  obj?: T;
}

export default function ClientQrModal({
  open,
  client,
  inboundsById,
  tunnelAllowedIPs,
  subSettings,
  hosts = NO_HOSTS,
  onOpenChange,
}: ClientQrModalProps) {
  const { t } = useTranslation();
  const [links, setLinks] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const publicHost = subSettings?.publicHost ?? '';

  const openEmail = open ? (client?.email ?? '') : '';
  const [syncedEmail, setSyncedEmail] = useState(openEmail);
  if (openEmail !== syncedEmail) {
    setSyncedEmail(openEmail);
    setLinks([]);
    setLoading(!!openEmail);
  }

  useEffect(() => {
    if (!openEmail) return;
    let cancelled = false;
    (async () => {
      try {
        const msg = (await HttpUtil.get(
          `/panel/api/clients/links/${encodeURIComponent(openEmail)}`,
        )) as ApiMsg<string[]>;
        if (!cancelled) {
          setLinks(msg?.success && Array.isArray(msg.obj) ? msg.obj : []);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [openEmail]);

  const host = window.location.hostname;
  const wgInbounds = useMemo(
    () => findWireguardInbounds(client, inboundsById),
    [client, inboundsById],
  );
  const wgConfigs = useMemo(() => {
    if (!client || !isWireguardClient(client)) return [];
    return wgInbounds
      .flatMap((ib) => {
        const address = tunnelAllowedIPs?.[ib.id] ?? '';
        return tunnelConfigEndpoints(ib, hosts, host, publicHost).map((ep) => ({
          inbound: ib,
          endpoint: tunnelEndpointLabel(ep),
          text: buildWireguardClientConfig(client, ib, host, publicHost, address, ep),
        }));
      })
      .filter((c) => !!c.text);
  }, [client, wgInbounds, tunnelAllowedIPs, publicHost, hosts, host]);

  const awgInbounds = useMemo(
    () => findAmneziaWGInbounds(client, inboundsById),
    [client, inboundsById],
  );
  const awgConfigs = useMemo(() => {
    if (!client || !isAmneziaWGClient(client)) return [];
    return awgInbounds
      .flatMap((ib) => {
        const address = tunnelAllowedIPs?.[ib.id] ?? '';
        return tunnelConfigEndpoints(ib, hosts, host, publicHost).map((ep) => ({
          inbound: ib,
          endpoint: tunnelEndpointLabel(ep),
          text: buildAmneziaWGClientConfig(client, ib, host, publicHost, address, ep),
        }));
      })
      .filter((c) => !!c.text);
  }, [client, awgInbounds, tunnelAllowedIPs, publicHost, hosts, host]);

  const tuicInbound = useMemo(() => findTuicInbound(client, inboundsById), [client, inboundsById]);
  const tuicConfigs = useMemo(() => {
    if (!client || !tuicInbound || !isTuicClient(client)) return [];
    return tunnelConfigEndpoints(tuicInbound, hosts, host, publicHost, 'clash').map((ep) => ({
      endpoint: tunnelEndpointLabel(ep),
      text: buildTuicClientConfig(client, tuicInbound, host, publicHost, ep),
    }));
  }, [client, tuicInbound, publicHost, hosts, host]);

  const items = useMemo(() => {
    const out: { key: string; label: React.ReactNode; children: React.ReactNode }[] = [];
    links.forEach((link, idx) => {
      const parts = parseLinkParts(link);
      const meta = parts ? linkMetaText(parts) : '';
      const label: React.ReactNode = parts ? (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
          <LinkTags parts={parts} />
          {meta && <span style={{ opacity: 0.6, fontSize: 12 }}>({meta})</span>}
        </span>
      ) : (
        `${t('pages.clients.link')} ${idx + 1}`
      );
      out.push({
        key: `l${idx}`,
        label,
        children: (
          <QrPanel
            value={link}
            remark={parts?.remark || `${client?.email || ''} #${idx + 1}`}
            showQr={!isPostQuantumLink(link)}
          />
        ),
      });
    });
    wgConfigs.forEach(({ inbound, endpoint, text }) => {
      const meta = formatTunnelConfigMeta(inbound, client?.email, wgConfigs.length, endpoint);
      out.push({
        key: `wg-config-${inbound.id}-${endpoint}`,
        label: (
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <Tag color="cyan" style={{ margin: 0 }}>
              {t('pages.clients.wireguardConfig')}
            </Tag>
            {meta.label && <span style={{ opacity: 0.85, fontSize: 12 }}>{meta.label}</span>}
          </span>
        ),
        children: <QrPanel value={text} remark={meta.qrRemark} downloadName={meta.fileName} />,
      });
    });
    awgConfigs.forEach(({ inbound, endpoint, text }) => {
      const meta = formatTunnelConfigMeta(inbound, client?.email, awgConfigs.length, endpoint);
      out.push({
        key: `awg-config-${inbound.id}-${endpoint}`,
        label: (
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <Tag color="purple" style={{ margin: 0 }}>
              {t('pages.clients.amneziaWgConfig')}
            </Tag>
            {meta.label && <span style={{ opacity: 0.85, fontSize: 12 }}>{meta.label}</span>}
          </span>
        ),
        children: <QrPanel value={text} remark={meta.qrRemark} downloadName={meta.fileName} />,
      });
    });
    tuicConfigs.forEach(({ endpoint, text }) => {
      const name = client?.email || 'tuic';
      const multi = tuicConfigs.length > 1 ? endpoint : '';
      out.push({
        key: `tuic-config-${endpoint}`,
        label: (
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <Tag color="orange" style={{ margin: 0 }}>
              {t('pages.clients.tuicConfig')}
            </Tag>
            {multi && <span style={{ opacity: 0.85, fontSize: 12 }}>{multi}</span>}
          </span>
        ),
        children: (
          <QrPanel
            value={text}
            remark={[name, multi].filter(Boolean).join(' - ')}
            downloadName={`${[name, multi.replace(/[^\w.-]+/g, '_')].filter(Boolean).join('-')}.yaml`}
          />
        ),
      });
    });
    return out;
  }, [links, client?.email, wgConfigs, awgConfigs, tuicConfigs, t]);

  const firstKey = open && items.length > 0 ? items[0].key : null;
  const [activeKey, setActiveKey] = useState<string[]>([]);
  const [syncedFirstKey, setSyncedFirstKey] = useState<string | null>(null);
  if (firstKey !== syncedFirstKey) {
    setSyncedFirstKey(firstKey);
    setActiveKey(firstKey ? [firstKey] : []);
  }

  return (
    <Modal
      open={open}
      title={client ? `${t('qrCode')} — ${client.email}` : t('qrCode')}
      footer={null}
      width={520}
      centered
      onCancel={() => onOpenChange(false)}
    >
      <Spin spinning={loading}>
        {!loading && items.length === 0 && (
          <div style={{ padding: 24, textAlign: 'center', opacity: 0.6 }}>
            {t('pages.clients.noLinks')}
          </div>
        )}
        {items.length > 0 && (
          <Collapse
            activeKey={activeKey}
            onChange={(keys) =>
              setActiveKey(typeof keys === 'string' ? [keys] : (keys as string[]))
            }
            items={items}
          />
        )}
      </Spin>
    </Modal>
  );
}
