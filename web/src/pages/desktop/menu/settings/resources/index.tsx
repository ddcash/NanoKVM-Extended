import { useEffect, useRef, useState } from 'react';
import { Divider, Progress, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import { getResources } from '@/api/vm.ts';
import type { Resources as ResourceData } from '@/api/vm.ts';

import { Processes } from './processes.tsx';

const REFRESH_MS = 3000;

function formatBytes(bytes: number) {
  if (!bytes) return '-';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value.toFixed(value >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatUptime(seconds: number) {
  if (!seconds) return '-';
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

// Amber and red thresholds. Memory runs high by design on this device, so it
// only turns red when genuinely close to exhaustion.
function barColor(percent: number, warn: number, danger: number) {
  if (percent >= danger) return '#dc2626';
  if (percent >= warn) return '#d97706';
  return '#2563eb';
}

export const Resources = () => {
  const { t } = useTranslation();

  const [data, setData] = useState<ResourceData | null>(null);
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    function refresh() {
      getResources()
        .then((rsp: any) => {
          if (rsp.code !== 0) return;
          setData(rsp.data);
        })
        .catch(() => {
          // A dropped poll is not worth surfacing; the next one will land.
        });
    }

    refresh();
    timer.current = setInterval(refresh, REFRESH_MS);

    return () => {
      if (timer.current) clearInterval(timer.current);
    };
  }, []);

  const memoryUsed = data ? data.memoryTotal - data.memoryAvailable : 0;
  const diskUsed = data ? data.diskTotal - data.diskFree : 0;

  return (
    <>
      <div className="text-base">{t('settings.resources.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-6">
        <div className="flex flex-col space-y-1">
          <div className="flex items-center justify-between">
            <span>{t('settings.resources.cpu')}</span>
            <span className="text-sm text-neutral-400">{data ? `${data.cpuPercent}%` : '-'}</span>
          </div>
          <Progress
            percent={data?.cpuPercent ?? 0}
            showInfo={false}
            strokeColor={barColor(data?.cpuPercent ?? 0, 75, 90)}
          />
        </div>

        <div className="flex flex-col space-y-1">
          <div className="flex items-center justify-between">
            <span>{t('settings.resources.memory')}</span>
            <span className="text-sm text-neutral-400">
              {data ? `${formatBytes(memoryUsed)} / ${formatBytes(data.memoryTotal)}` : '-'}
            </span>
          </div>
          <Progress
            percent={data?.memoryPercent ?? 0}
            showInfo={false}
            strokeColor={barColor(data?.memoryPercent ?? 0, 85, 95)}
          />
          <span className="text-xs text-neutral-500">{t('settings.resources.memoryNote')}</span>
        </div>

        <div className="flex flex-col space-y-1">
          <div className="flex items-center justify-between">
            <span>{t('settings.resources.disk')}</span>
            <span className="text-sm text-neutral-400">
              {data ? `${formatBytes(diskUsed)} / ${formatBytes(data.diskTotal)}` : '-'}
            </span>
          </div>
          <Progress
            percent={data?.diskPercent ?? 0}
            showInfo={false}
            strokeColor={barColor(data?.diskPercent ?? 0, 80, 92)}
          />
        </div>
      </div>

      <Divider className="opacity-50" style={{ margin: '28px 0' }} />

      <div className="flex flex-col space-y-4">
        <div className="flex items-center justify-between">
          <span>{t('settings.resources.temperature')}</span>
          <span className="text-sm text-neutral-400">
            {data?.temperature ? `${data.temperature} °C` : '-'}
          </span>
        </div>

        <div className="flex items-center justify-between">
          <Tooltip title={t('settings.resources.loadTip')}>
            <span className="cursor-help border-b border-dotted border-neutral-600">
              {t('settings.resources.load')}
            </span>
          </Tooltip>
          <span className="text-sm text-neutral-400">
            {data ? `${data.load1} · ${data.load5} · ${data.load15}` : '-'}
          </span>
        </div>

        <div className="flex items-center justify-between">
          <span>{t('settings.resources.uptime')}</span>
          <span className="text-sm text-neutral-400">
            {data ? formatUptime(data.uptimeSeconds) : '-'}
          </span>
        </div>
      </div>

      <Divider className="opacity-50" style={{ margin: '28px 0' }} />

      <Processes />
    </>
  );
};
