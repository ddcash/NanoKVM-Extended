import { useEffect, useState } from 'react';
import { Button, message, Popconfirm, Table, Tag, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import { getProcesses, killProcess } from '@/api/vm.ts';
import type { ProcessInfo } from '@/api/vm.ts';

function formatBytes(bytes: number) {
  if (!bytes) return '-';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value.toFixed(value >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

export const Processes = () => {
  const { t } = useTranslation();

  const [processes, setProcesses] = useState<ProcessInfo[]>([]);
  const [busy, setBusy] = useState(0);

  useEffect(() => {
    refresh();
  }, []);

  function refresh() {
    getProcesses()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setProcesses(rsp.data?.processes ?? []);
      })
      .catch(() => {
        // A dropped poll is not worth surfacing.
      });
  }

  function kill(process: ProcessInfo, force: boolean) {
    setBusy(process.pid);

    killProcess(process.pid, force)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.resources.killFailed'));
          return;
        }
        message.success(t('settings.resources.killed', { name: process.name }));
        // A SIGTERM'd process may take a moment to go away.
        setTimeout(refresh, 800);
      })
      .catch(() => message.error(t('settings.resources.killFailed')))
      .finally(() => setBusy(0));
  }

  const columns = [
    {
      title: t('settings.resources.process'),
      dataIndex: 'name',
      render: (_: string, row: ProcessInfo) => (
        <Tooltip title={row.command || row.name}>
          <span className="text-sm">{row.name}</span>
        </Tooltip>
      )
    },
    {
      title: 'PID',
      dataIndex: 'pid',
      width: 80,
      render: (pid: number) => <span className="text-xs text-neutral-500">{pid}</span>
    },
    {
      title: t('settings.resources.memory'),
      dataIndex: 'memoryBytes',
      width: 110,
      render: (bytes: number, row: ProcessInfo) => (
        <span className="text-xs text-neutral-400">
          {formatBytes(bytes)}
          {row.memoryPercent > 0 && ` (${row.memoryPercent}%)`}
        </span>
      )
    },
    {
      title: '',
      dataIndex: 'pid',
      width: 130,
      render: (_: number, row: ProcessInfo) =>
        row.protected ? (
          <Tooltip title={t('settings.resources.protectedTip')}>
            <Tag>{t('settings.resources.protected')}</Tag>
          </Tooltip>
        ) : (
          <div className="flex space-x-1">
            <Popconfirm
              title={t('settings.resources.stopConfirm', { name: row.name })}
              onConfirm={() => kill(row, false)}
            >
              <Button size="small" loading={busy === row.pid}>
                {t('settings.resources.stop')}
              </Button>
            </Popconfirm>
            <Popconfirm
              title={t('settings.resources.forceConfirm', { name: row.name })}
              onConfirm={() => kill(row, true)}
            >
              <Button size="small" danger>
                {t('settings.resources.force')}
              </Button>
            </Popconfirm>
          </div>
        )
    }
  ];

  return (
    <div className="flex flex-col space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.resources.processes')}</span>
          <span className="text-xs text-neutral-500">{t('settings.resources.processesDesc')}</span>
        </div>
        <Button onClick={refresh}>{t('settings.resources.refresh')}</Button>
      </div>

      <Table
        size="small"
        rowKey="pid"
        columns={columns}
        dataSource={processes}
        pagination={false}
        scroll={{ y: 320 }}
      />
    </div>
  );
};
