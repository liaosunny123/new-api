import React, { useState, useCallback, useMemo } from 'react';
import {
  Button,
  InputNumber,
  Select,
  Tag,
  Typography,
  Popconfirm,
} from '@douyinfe/semi-ui';
import { IconPlus, IconDelete } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

function parseJSON(str) {
  if (!str || !str.trim()) return {};
  try {
    return JSON.parse(str);
  } catch {
    return {};
  }
}

function serializeGroupRateLimit(entries) {
  const result = {};
  entries.forEach(({ group, rpm, concurrency }) => {
    if (!group) return;
    result[group] = { rpm: rpm || 0, concurrency: concurrency || 0 };
  });
  return Object.keys(result).length === 0
    ? ''
    : JSON.stringify(result, null, 2);
}

let _idCounter = 0;
const uid = () => `grl_${++_idCounter}`;

export default function GroupRateLimitRules({
  value,
  groupNames = [],
  onChange,
}) {
  const { t } = useTranslation();
  const [entries, setEntries] = useState(() => {
    const parsed = parseJSON(value);
    return Object.entries(parsed).map(([group, cfg]) => ({
      _id: uid(),
      group,
      rpm: cfg.rpm || 0,
      concurrency: cfg.concurrency || 0,
    }));
  });
  const [newGroupName, setNewGroupName] = useState('');

  const emitChange = useCallback(
    (newEntries) => {
      setEntries(newEntries);
      onChange?.(serializeGroupRateLimit(newEntries));
    },
    [onChange],
  );

  const updateEntry = useCallback(
    (id, field, val) => {
      emitChange(
        entries.map((e) => (e._id === id ? { ...e, [field]: val } : e)),
      );
    },
    [entries, emitChange],
  );

  const removeEntry = useCallback(
    (id) => {
      emitChange(entries.filter((e) => e._id !== id));
    },
    [entries, emitChange],
  );

  const addEntry = useCallback(() => {
    const name = newGroupName.trim();
    if (!name) return;
    if (entries.some((e) => e.group === name)) return;
    emitChange([
      ...entries,
      { _id: uid(), group: name, rpm: 0, concurrency: 0 },
    ]);
    setNewGroupName('');
  }, [entries, emitChange, newGroupName]);

  const groupOptions = useMemo(
    () => groupNames.map((n) => ({ value: n, label: n })),
    [groupNames],
  );

  return (
    <div>
      <Text type='tertiary' size='small' style={{ display: 'block', marginBottom: 8 }}>
        {t('RPM 为 0 表示不限制，并发数为 0 表示不限制')}
      </Text>
      {entries.map((entry) => (
        <div
          key={entry._id}
          className='flex items-center gap-2'
          style={{ marginBottom: 8 }}
        >
          <Tag size='large' color='blue' style={{ minWidth: 80 }}>
            {entry.group}
          </Tag>
          <Text size='small' style={{ whiteSpace: 'nowrap' }}>RPM:</Text>
          <InputNumber
            size='small'
            min={0}
            step={1}
            value={entry.rpm}
            style={{ width: 100 }}
            onChange={(v) => updateEntry(entry._id, 'rpm', v ?? 0)}
          />
          <Text size='small' style={{ whiteSpace: 'nowrap' }}>{t('并发')}:</Text>
          <InputNumber
            size='small'
            min={0}
            step={1}
            value={entry.concurrency}
            style={{ width: 100 }}
            onChange={(v) => updateEntry(entry._id, 'concurrency', v ?? 0)}
          />
          <Popconfirm
            title={t('确认删除该规则？')}
            onConfirm={() => removeEntry(entry._id)}
            position='left'
          >
            <Button
              icon={<IconDelete />}
              type='danger'
              theme='borderless'
              size='small'
            />
          </Popconfirm>
        </div>
      ))}
      <div className='mt-2 flex items-center gap-2'>
        <Select
          size='small'
          filter
          allowCreate
          placeholder={t('选择分组')}
          optionList={groupOptions}
          value={newGroupName || undefined}
          onChange={setNewGroupName}
          style={{ width: 200 }}
          position='bottomLeft'
        />
        <Button icon={<IconPlus />} theme='outline' size='small' onClick={addEntry}>
          {t('添加分组限制')}
        </Button>
      </div>
    </div>
  );
}
